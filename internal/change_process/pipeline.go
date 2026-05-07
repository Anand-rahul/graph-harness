// Package change_process implements the 12-stage validation pipeline.
// Phase 0 emits exactly one finding kind: `flow_unreviewed`. Empty repair
// payload until P3 ships RepairInstruction synthesis. The pipeline is
// idempotent at a fixed kernel sequence (SPEC §8.1 + plan §P0.T28-T36).
package change_process

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/shivamstaq/graph-harness/internal/code_core"
	"github.com/shivamstaq/graph-harness/internal/semantic_overlay"
)

// ValidationFinding is the SPEC §8.2 shape (Phase 0 fields). The CLI
// emits this as JSON via `validate-diff --json`.
type ValidationFinding struct {
	ID       string         `json:"id"`
	Kind     string         `json:"kind"`     // P0: only "flow_unreviewed"
	Severity string         `json:"severity"` // info | low | medium | high | critical
	Subject  Subject        `json:"subject"`
	Evidence []EvidenceItem `json:"evidence"`
	Repair   map[string]any `json:"repair"` // empty in P0, populated in P3
}

// Subject identifies the entity the finding is about (selector resolution).
type Subject struct {
	EntityKind string `json:"entity_kind"`
	EntityID   string `json:"entity_id"`
	Qualified  string `json:"qualified_name"`
	Flow       string `json:"flow,omitempty"`
}

// EvidenceItem documents one piece of evidence supporting the finding.
type EvidenceItem struct {
	Kind   string `json:"kind"`   // e.g. "flow_touched_without_ack"
	Detail string `json:"detail"` // free-form English
}

// ValidateDiffResult summarizes a single validate-diff invocation.
type ValidateDiffResult struct {
	ValidationSeq uint64              `json:"validation_seq"`
	DiffSHA       string              `json:"diff_sha"`
	Findings      []ValidationFinding `json:"findings"`
	Summary       string              `json:"summary"`
}

// Pipeline wires together the kernel state required to validate a diff.
type Pipeline struct {
	Overlay *semantic_overlay.Overlay
	Code    *code_core.Store
}

// ValidateDiff runs all 12 stages against the unified diff in `unified` and
// returns the structured result. Idempotent at fixed seq.
func (p *Pipeline) ValidateDiff(ctx context.Context, unified []byte, validationSeq uint64) (*ValidateDiffResult, error) {
	res := &ValidateDiffResult{
		ValidationSeq: validationSeq,
		DiffSHA:       hashBytes(unified),
		Findings:      []ValidationFinding{},
	}

	// Stage 1: parse diff → hunks per file.
	hunks := parseUnifiedDiff(unified)
	if len(hunks) == 0 {
		res.Summary = "0 findings (empty diff)"
		return res, nil
	}

	// Stage 2: map hunks → code.core entities (P0 = qualified-name lookup
	// against modified function lines). Production resolution requires
	// tree-sitter range overlap (P0.T29 fully); the qualified-name path
	// covers the demo end-to-end and graduates as the indexer matures.
	touched := map[string]string{} // qualified_name -> entity_id
	for _, h := range hunks {
		for _, qn := range extractFunctionNames(h.AddedLines) {
			ent, err := p.Code.LookupByQualifiedNameSuffix(ctx, qn)
			if err != nil {
				return nil, err
			}
			if ent != nil {
				touched[ent.QualifiedName] = ent.ID
			}
		}
	}

	// Stages 3, 3b: bounded refresh + pin validation_seq — implicit at the
	// validation_seq input here. We honor the contract.
	// Stages 4, 5: touched_set + impacted_set (P0 = touched only; impacted
	// requires call-graph edges that arrive in P1 via LSP/SCIP).
	// Stage 6: resolve flow selectors against touched.
	for flowName, flow := range p.Overlay.Flows {
		// In P0 we resolve the flow's scope as a selector by name.
		scope := flow.Scope
		if scope == "" {
			continue
		}
		env, err := p.Overlay.Resolve(ctx, scope, p.Code, validationSeq)
		if err != nil {
			continue
		}
		for _, m := range env.Matches {
			if _, hit := touched[m.QualifiedName]; !hit {
				continue
			}
			// Stage 7-9: invariant/control/skill checks (pass-through in P0).
			// Stage 10: emit the finding.
			f := ValidationFinding{
				ID:       newFindingID(res.DiffSHA, flowName, m.QualifiedName),
				Kind:     "flow_unreviewed",
				Severity: "medium",
				Subject: Subject{
					EntityKind: "code.core:Function",
					EntityID:   m.EntityID,
					Qualified:  m.QualifiedName,
					Flow:       flowName,
				},
				Evidence: []EvidenceItem{{
					Kind:   "flow_touched_without_ack",
					Detail: fmt.Sprintf("touched flow-scoped function %s without acknowledging flow %s", m.QualifiedName, flowName),
				}},
				Repair: map[string]any{}, // empty in P0; P3 populates
			}
			res.Findings = append(res.Findings, f)
		}
	}

	// Stage 11: pass-through. Stage 12: emit the summary.
	res.Summary = fmt.Sprintf("%d findings", len(res.Findings))
	return res, nil
}

// Hunk is a single hunk extracted from a unified diff, keyed by file path.
type Hunk struct {
	Path       string
	AddedLines []string
}

var hunkHeaderRE = regexp.MustCompile(`^@@`)

func parseUnifiedDiff(b []byte) []Hunk {
	lines := strings.Split(string(b), "\n")
	var hunks []Hunk
	var current Hunk
	flush := func() {
		if current.Path != "" || len(current.AddedLines) > 0 {
			hunks = append(hunks, current)
		}
		current = Hunk{}
	}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++ "):
			flush()
			current.Path = strings.TrimPrefix(strings.TrimSpace(line), "+++ ")
			// Strip the leading "b/" prefix git emits.
			current.Path = strings.TrimPrefix(current.Path, "b/")
		case hunkHeaderRE.MatchString(line):
			// Hunk delimiter; keep accumulating into current.
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			current.AddedLines = append(current.AddedLines, strings.TrimPrefix(line, "+"))
		}
	}
	flush()
	return hunks
}

// extractFunctionNames grabs `func Name(` and `func (recv *T) Method(`
// signatures out of added lines. The names returned are package-less
// (resolution against code.core handles namespacing via the qualified_name
// column, which preserves package prefix when the file was ingested).
//
// In P0 the matcher operates by suffix: the qualified_name column carries
// `pkg.Name` or `pkg.Recv.Name`, so we match on the trailing component.
var (
	funcRE   = regexp.MustCompile(`^\s*func\s+([A-Z][A-Za-z0-9_]*)\s*\(`)
	methodRE = regexp.MustCompile(`^\s*func\s+\([^)]*\)\s+([A-Z][A-Za-z0-9_]*)\s*\(`)
)

func extractFunctionNames(addedLines []string) []string {
	out := []string{}
	for _, l := range addedLines {
		if m := methodRE.FindStringSubmatch(l); m != nil {
			out = append(out, m[1])
			continue
		}
		if m := funcRE.FindStringSubmatch(l); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

func hashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])[:16]
}

func newFindingID(diffSha, flowName, qn string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(diffSha + "\x00" + flowName + "\x00" + qn))
	return "finding_" + hex.EncodeToString(h.Sum(nil))[:8]
}
