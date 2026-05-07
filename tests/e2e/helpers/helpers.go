// Package helpers provides graph-harness-specific gotit registrations:
// repo helpers, requirement checkers, and custom assertion types.
//
// Helpers stay small and exemplary — they are the authoritative builders
// of E2E fixtures. Anything project-specific belongs here, not in the
// gotit kit.
package helpers

import (
	"errors"
	"os/exec"

	"github.com/shivamstaq/gotit/runner"
)

// CheckCGO verifies that a C toolchain is available — required for tree-sitter.
func CheckCGO(_ string) error {
	if _, err := exec.LookPath("cc"); err != nil {
		if _, err2 := exec.LookPath("gcc"); err2 != nil {
			return errors.New("no C compiler on PATH (need cc or gcc for cgo)")
		}
	}
	return nil
}

// CheckTreeSitter verifies tree-sitter cgo bindings link by checking that
// the just-built graph-harness binary can spawn a parse via the smoke command.
// (Until the smoke command exists, we proxy via cgo availability.)
func CheckTreeSitter(_ string) error {
	return CheckCGO("")
}

// Compile-time assertion that we satisfy the runner interfaces.
var (
	_ runner.RepoHelper         = GoModuleEmpty
	_ runner.RepoHelper         = GoModuleWithCheckoutValidator
	_ runner.RequirementChecker = CheckCGO
	_ runner.RequirementChecker = CheckTreeSitter
)
