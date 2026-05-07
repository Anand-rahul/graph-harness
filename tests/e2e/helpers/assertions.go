package helpers

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/shivamstaq/gotit/runner"
)

// AssertEventLogMonotonic reads the workspace's SQLite event log and asserts
// that kernel_events.seq is strictly monotonically increasing across all rows.
//
// YAML usage:
//
//   - type: x-event-log-monotonic
//     path: ".graph-harness/state.db"   # relative to workDir; default if empty
//
// The runner's <PREFIX>_E2E_PROJECT_ROOT env var is the consumer's repo root,
// not the spec's workDir, so callers pass the workspace-relative path here.
// We detect workDir from the runner's exec environment via stdout convention:
// the first call reads the path from result.Stdout if path is empty.
func AssertEventLogMonotonic(a runner.Assertion, r runner.StepResult, _ map[string]string) error {
	dbPath := a.Path
	if dbPath == "" {
		// Convention: a preceding step prints the absolute path of the event log
		// to stdout (e.g. via `graph-harness daemon db-path`).
		dbPath = filepath.Clean(r.Stdout)
	}
	if dbPath == "" {
		return fmt.Errorf("x-event-log-monotonic: path is empty and stdout did not carry a path")
	}
	// Use sqlite3 CLI if available; falls back to a clear error otherwise.
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return fmt.Errorf("x-event-log-monotonic: sqlite3 CLI not on PATH (install or use a Go-side helper)")
	}
	// #nosec G204 -- dbPath is sourced from the spec/test fixture, not user input.
	out, err := exec.Command("sqlite3", dbPath,
		"SELECT seq FROM kernel_events ORDER BY ROWID").CombinedOutput()
	if err != nil {
		return fmt.Errorf("x-event-log-monotonic: sqlite3 query failed: %v: %s", err, out)
	}
	var prev int64 = -1
	for _, line := range splitLines(string(out)) {
		if line == "" {
			continue
		}
		var seq int64
		if _, err := fmt.Sscanf(line, "%d", &seq); err != nil {
			return fmt.Errorf("x-event-log-monotonic: bad seq row %q: %v", line, err)
		}
		if seq <= prev {
			return fmt.Errorf("x-event-log-monotonic: non-monotonic at seq %d (prev %d)", seq, prev)
		}
		prev = seq
	}
	return nil
}

// AssertFindingShapeValid validates a JSON ValidationFinding (or array of them)
// against the SPEC §8.2 shape. The finding text comes from r.Stdout.
//
//   - type: x-finding-shape-valid
//     path: "$[0]"   # optional JSONPath; default = whole stdout
func AssertFindingShapeValid(a runner.Assertion, r runner.StepResult, _ map[string]string) error {
	var raw any
	if err := json.Unmarshal([]byte(r.Stdout), &raw); err != nil {
		return fmt.Errorf("x-finding-shape-valid: stdout is not valid JSON: %v", err)
	}
	target := raw
	if a.Path != "" {
		// gotit ships ojg; for simplicity here we only support the common
		// "$[<n>]" array-index form used by our specs.
		// A future iteration will use ojg/jp for full JSONPath support.
		var idx int
		if _, err := fmt.Sscanf(a.Path, "$[%d]", &idx); err != nil {
			return fmt.Errorf("x-finding-shape-valid: only $[N] paths supported, got %q", a.Path)
		}
		arr, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("x-finding-shape-valid: stdout is not an array")
		}
		if idx >= len(arr) {
			return fmt.Errorf("x-finding-shape-valid: index %d out of range (len %d)", idx, len(arr))
		}
		target = arr[idx]
	}
	finding, ok := target.(map[string]any)
	if !ok {
		return fmt.Errorf("x-finding-shape-valid: target is not a JSON object")
	}
	required := []string{"id", "kind", "severity", "subject", "evidence"}
	for _, k := range required {
		if _, ok := finding[k]; !ok {
			return fmt.Errorf("x-finding-shape-valid: missing required field %q (SPEC §8.2)", k)
		}
	}
	// `repair` is required but may be empty in P0 (P3 populates it).
	if _, ok := finding["repair"]; !ok {
		return fmt.Errorf("x-finding-shape-valid: missing repair field (may be empty in P0, but field must exist)")
	}
	return nil
}

// AssertManifestValid pipes a YAML manifest through `graph-harness layers
// install --dry-run` and asserts a clean exit. Used to confirm a generated
// manifest passes the validator without committing it.
//
//   - type: x-manifest-valid
//     path: ".graph-harness/manifests/foo.yaml"   # path relative to workDir
func AssertManifestValid(a runner.Assertion, _ runner.StepResult, _ map[string]string) error {
	if a.Path == "" {
		return fmt.Errorf("x-manifest-valid: path is required")
	}
	// #nosec G204 -- a.Path is sourced from the spec, an authored test artifact.
	cmd := exec.Command("graph-harness", "layers", "install", "--dry-run", a.Path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("x-manifest-valid: validator rejected %s: %v\n%s", a.Path, err, out)
	}
	return nil
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
