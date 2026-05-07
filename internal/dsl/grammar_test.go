package dsl

import (
	"strings"
	"testing"
)

func TestParse_SelectorWithSingleAnchor(t *testing.T) {
	src := `
selector CheckoutValidator {
  unique
  anchor qualified_name "CheckoutValidator.Validate"
}
`
	f, err := ParseString("test.gh", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Decls) != 1 || f.Decls[0].Selector == nil {
		t.Fatalf("expected one selector decl, got %+v", f.Decls)
	}
	s := f.Decls[0].Selector
	if s.Name != "CheckoutValidator" {
		t.Errorf("name = %q", s.Name)
	}
	if !s.Unique {
		t.Errorf("unique flag not set")
	}
	if len(s.Anchors) != 1 || s.Anchors[0].Kind != "qualified_name" {
		t.Errorf("anchors = %+v", s.Anchors)
	}
}

func TestParse_FlowWithStep(t *testing.T) {
	src := `
flow CheckoutValidation {
  description "Pre-payment cart validation"
  scope CheckoutValidator
  step ValidateCart targets selector { qualified_name "CheckoutValidator.Validate" }
}
`
	f, err := ParseString("test.gh", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	flow := f.Decls[0].Flow
	if flow == nil {
		t.Fatalf("expected flow, got %+v", f.Decls)
	}
	if flow.Name != "CheckoutValidation" {
		t.Errorf("name = %q", flow.Name)
	}
	if len(flow.Steps) != 1 || flow.Steps[0].Name != "ValidateCart" {
		t.Errorf("steps = %+v", flow.Steps)
	}
}

func TestParse_RejectsDatalogRule(t *testing.T) {
	src := `rule reaches_payment(F) :- entity(F, "Function").`
	_, err := ParseString("test.gh", src)
	if err == nil || !strings.Contains(err.Error(), "datalog rules deferred") {
		t.Fatalf("expected Datalog rejection, got %v", err)
	}
}

func TestRoundtrip_SelectorAndFlow(t *testing.T) {
	src := `selector CheckoutValidator {
  unique
  anchor qualified_name "CheckoutValidator.Validate"
}

flow CheckoutValidation {
  description "Pre-payment cart validation"
  scope CheckoutValidator
  step ValidateCart targets selector { qualified_name "CheckoutValidator.Validate" }
}
`
	f, err := ParseString("test.gh", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rendered := Render(f)
	f2, err := ParseString("rendered.gh", rendered)
	if err != nil {
		t.Fatalf("re-parse rendered: %v\n%s", err, rendered)
	}
	if len(f.Decls) != len(f2.Decls) {
		t.Fatalf("decl count differs: orig=%d, rendered=%d", len(f.Decls), len(f2.Decls))
	}
	if f.Decls[0].Selector.Name != f2.Decls[0].Selector.Name {
		t.Errorf("selector name diverged after roundtrip")
	}
	if f.Decls[1].Flow.Name != f2.Decls[1].Flow.Name {
		t.Errorf("flow name diverged after roundtrip")
	}
}

func TestParse_Comments(t *testing.T) {
	src := `
// top comment
selector S { /* inline */ unique anchor qualified_name "X" }
`
	f, err := ParseString("test.gh", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if f.Decls[0].Selector.Name != "S" {
		t.Errorf("comment elision broke parse: %+v", f.Decls)
	}
}
