package helpers

import (
	"path/filepath"

	"github.com/shivamstaq/gotit/runner"
)

// GoModuleEmpty creates a workspace with `go.mod` only. Use for `init` and
// daemon-lifecycle specs that don't care about source code.
func GoModuleEmpty(_, workDir string, _ map[string]any) error {
	if err := runner.WriteFile(
		filepath.Join(workDir, "go.mod"),
		"module example.com/empty\n\ngo 1.23\n",
	); err != nil {
		return err
	}
	if err := runner.GitInit(workDir); err != nil {
		return err
	}
	return runner.GitCommitAll(workDir, "initial empty go module")
}

// GoModuleWithCheckoutValidator creates a small Go module containing a
// CheckoutValidator type with a Validate method, plus a callsite, plus the
// canonical .gh overlay flow that the tracer demo exercises. This is the
// Phase-0 demo fixture.
func GoModuleWithCheckoutValidator(_, workDir string, _ map[string]any) error {
	overlayGH := `selector CheckoutValidator {
  unique
  anchor qualified_name "checkout.CheckoutValidator.Validate"
}

flow CheckoutValidation {
  description "Pre-payment cart validation"
  scope CheckoutValidator
  step ValidateCart targets selector { qualified_name "checkout.CheckoutValidator.Validate" }
}
`
	diff := `--- a/internal/checkout/validator.go
+++ b/internal/checkout/validator.go
@@ -3,5 +3,7 @@ package checkout
 type CheckoutValidator struct{}

-func (v *CheckoutValidator) Validate(cart any) error {
+// Validate runs the cart validation pipeline (now with extra logging).
+func (v *CheckoutValidator) Validate(cart any) error {
+	_ = cart
 	return nil
 }
`
	files := map[string]string{
		"go.mod": "module example.com/checkout\n\ngo 1.23\n",
		"internal/checkout/validator.go": `package checkout

// CheckoutValidator validates pre-payment cart state.
type CheckoutValidator struct{}

// Validate runs the cart validation pipeline.
func (v *CheckoutValidator) Validate(cart any) error {
	if cart == nil {
		return nil
	}
	return nil
}
`,
		"main.go": `package main

import (
	"example.com/checkout/internal/checkout"
)

func main() {
	v := &checkout.CheckoutValidator{}
	_ = v.Validate(nil)
}
`,
	}
	for relPath, content := range files {
		if err := runner.WriteFile(filepath.Join(workDir, relPath), content); err != nil {
			return err
		}
	}
	if err := runner.GitInit(workDir); err != nil {
		return err
	}
	if err := runner.GitCommitAll(workDir, "initial checkout fixture"); err != nil {
		return err
	}
	// Pre-stage the demo .gh overlay AND the target diff in the workspace
	// (outside .graph-harness/, which `init` will create). The spec consumes
	// these without needing heredoc escapes in YAML.
	if err := runner.WriteFile(filepath.Join(workDir, "demo.checkout.gh"), overlayGH); err != nil {
		return err
	}
	return runner.WriteFile(filepath.Join(workDir, "demo.diff"), diff)
}
