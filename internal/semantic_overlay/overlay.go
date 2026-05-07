// Package semantic_overlay implements the semantic.overlay layer importer
// and the single-anchor selector resolver. SPEC §3, §11. Phase 0 single
// anchor + outcomes {bound, unresolved}; multi-anchor + 5-outcome lands in P3.
package semantic_overlay

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shivamstaq/graph-harness/internal/code_core"
	"github.com/shivamstaq/graph-harness/internal/dsl"
)

// Overlay is the parsed, in-memory view of every .gh file in the workspace.
// SPEC §4.8: the .gh files on disk are canonical; the overlay index is
// rebuildable from source.
type Overlay struct {
	Selectors map[string]*dsl.Selector
	Flows     map[string]*dsl.Flow
}

// NewOverlay returns an empty overlay.
func NewOverlay() *Overlay {
	return &Overlay{
		Selectors: map[string]*dsl.Selector{},
		Flows:     map[string]*dsl.Flow{},
	}
}

// Load walks the overlay directory (.graph-harness/overlay/**) and parses
// every .gh file into the overlay. Errors are returned per-file via the
// returned error map; the overlay still contains everything that parsed.
func (o *Overlay) Load(overlayDir string) (map[string]error, error) {
	errs := map[string]error{}
	if _, err := os.Stat(overlayDir); os.IsNotExist(err) {
		return errs, nil // no overlay yet is fine
	}
	walkErr := filepath.WalkDir(overlayDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".gh") {
			return nil
		}
		// #nosec G304,G122 -- overlay directory is operator-controlled and rooted.
		data, err := os.ReadFile(path) //nolint:gosec
		if err != nil {
			errs[path] = err
			return nil
		}
		f, err := dsl.ParseString(path, string(data))
		if err != nil {
			errs[path] = err
			return nil
		}
		for _, decl := range f.Decls {
			switch {
			case decl.Selector != nil:
				o.Selectors[decl.Selector.Name] = decl.Selector
			case decl.Flow != nil:
				o.Flows[decl.Flow.Name] = decl.Flow
			}
		}
		return nil
	})
	return errs, walkErr
}

// ResolutionOutcome enumerates the P0 subset of SPEC §3.2 outcomes.
// Multi-anchor outcomes (reanchored, ambiguous, superseded) land in P3.
type ResolutionOutcome string

// Phase 0 resolution outcomes (SPEC §3.2 subset).
const (
	OutcomeBound      ResolutionOutcome = "bound"
	OutcomeUnresolved ResolutionOutcome = "unresolved"
)

// ResolutionMatch is one resolved entity reference.
type ResolutionMatch struct {
	EntityID      string  `json:"entity_id"`
	QualifiedName string  `json:"qualified_name"`
	Confidence    float64 `json:"confidence"`
	ViaAnchor     string  `json:"via_anchor"`
}

// ResolutionEnvelope is the structured result returned by selector resolution
// (SPEC §3.2). v0 envelope; multi-anchor extras land in P3.
type ResolutionEnvelope struct {
	SelectorID string            `json:"selector_id"`
	Outcome    ResolutionOutcome `json:"outcome"`
	Matches    []ResolutionMatch `json:"matches"`
	ResolvedAt uint64            `json:"resolved_at_kernel_seq"`
}

// Resolve runs the v0 single-anchor resolver against the code.core store.
// Looks up the qualified_name anchor; everything else is "unresolved" in P0.
func (o *Overlay) Resolve(ctx context.Context, name string, store *code_core.Store, atSeq uint64) (*ResolutionEnvelope, error) {
	sel, ok := o.Selectors[name]
	if !ok {
		return nil, fmt.Errorf("selector %q not found in overlay", name)
	}
	env := &ResolutionEnvelope{
		SelectorID: sel.Name,
		Outcome:    OutcomeUnresolved,
		ResolvedAt: atSeq,
	}
	for _, anchor := range sel.Anchors {
		if anchor.Kind != "qualified_name" || anchor.Value == nil || anchor.Value.Str == nil {
			continue
		}
		qn := *anchor.Value.Str
		ent, err := store.LookupByQualifiedName(ctx, qn)
		if err != nil {
			return nil, err
		}
		if ent != nil {
			env.Outcome = OutcomeBound
			env.Matches = append(env.Matches, ResolutionMatch{
				EntityID:      ent.ID,
				QualifiedName: ent.QualifiedName,
				Confidence:    0.97, // single-anchor exact match in P0
				ViaAnchor:     "qualified_name",
			})
			break
		}
	}
	return env, nil
}
