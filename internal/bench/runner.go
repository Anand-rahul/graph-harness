// Package bench implements the bench.scenarios + bench.oracle layers and the
// `graph-harness bench` runner. P0.T46-T48: scenario 1 only (auth-sensitive
// edit, Go), mature regime, detection axis only.
package bench

import (
	"encoding/json"

	"github.com/shivamstaq/graph-harness/internal/change_process"
)

// Scenario describes one bench fixture (SPEC §12).
type Scenario struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Regime      string   `json:"regime"`
	Description string   `json:"description"`
	Languages   []string `json:"languages"`
}

// ExpectedFinding is the oracle ground-truth shape (SPEC §12).
type ExpectedFinding struct {
	Kind         string `json:"kind"`
	Severity     string `json:"severity"`
	SubjectQName string `json:"subject_qualified_name"`
	Flow         string `json:"flow"`
}

// AxisScore tracks one of the four axes from SPEC §12.
type AxisScore struct {
	Score    float64 `json:"score"`
	Notes    string  `json:"notes,omitempty"`
	Detected int     `json:"detected,omitempty"`
	Expected int     `json:"expected,omitempty"`
}

// Result is the per-scenario score envelope.
type Result struct {
	Scenario        Scenario  `json:"scenario"`
	DetectionAxis   AxisScore `json:"detection_axis"`
	RepairQuality   AxisScore `json:"repair_quality"`
	RepairExecution AxisScore `json:"repair_execution"`
	FinalCleanPatch AxisScore `json:"final_clean_patch"`
	Findings        []string  `json:"findings_observed"`
	Expected        []string  `json:"findings_expected"`
}

// Score compares a pipeline result against the oracle.
func Score(scenario Scenario, observed *change_process.ValidateDiffResult, expected []ExpectedFinding) *Result {
	res := &Result{
		Scenario: scenario,
		DetectionAxis: AxisScore{
			Score:    0,
			Expected: len(expected),
		},
		RepairQuality:   AxisScore{Score: -1, Notes: "N/A in P0 (deferred to P3)"},
		RepairExecution: AxisScore{Score: -1, Notes: "N/A in P0 (deferred to P3)"},
		FinalCleanPatch: AxisScore{Score: -1, Notes: "N/A in P0 (deferred to P3)"},
	}
	matched := 0
	for _, want := range expected {
		for _, got := range observed.Findings {
			if got.Kind == want.Kind && got.Subject.Flow == want.Flow {
				matched++
				break
			}
		}
	}
	res.DetectionAxis.Detected = matched
	if len(expected) > 0 {
		res.DetectionAxis.Score = float64(matched) / float64(len(expected))
	} else if len(observed.Findings) == 0 {
		res.DetectionAxis.Score = 1.0
	}
	for _, f := range observed.Findings {
		res.Findings = append(res.Findings, f.Kind+":"+f.Subject.Flow)
	}
	for _, e := range expected {
		res.Expected = append(res.Expected, e.Kind+":"+e.Flow)
	}
	return res
}

// AsJSON renders a Result as pretty JSON for CLI output.
func (r *Result) AsJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
