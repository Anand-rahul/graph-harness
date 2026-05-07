package source_live

import "testing"

func TestParseGoFile_ExtractsFunctionsAndMethods(t *testing.T) {
	src := []byte(`package checkout

// CheckoutValidator validates pre-payment cart state.
type CheckoutValidator struct{}

// Validate runs the cart validation pipeline.
func (v *CheckoutValidator) Validate(cart any) error {
	return nil
}

func helper() string { return "hi" }
`)
	pf, err := ParseGoFile("validator.go", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if pf.Language != "go" {
		t.Errorf("language = %q, want go", pf.Language)
	}
	if len(pf.Functions) != 2 {
		t.Fatalf("got %d functions, want 2: %+v", len(pf.Functions), pf.Functions)
	}
	want := map[string]string{
		"checkout.CheckoutValidator.Validate": "CheckoutValidator",
		"checkout.helper":                     "",
	}
	for _, fn := range pf.Functions {
		recv, ok := want[fn.QualifiedName]
		if !ok {
			t.Errorf("unexpected function %q", fn.QualifiedName)
			continue
		}
		if fn.Receiver != recv {
			t.Errorf("receiver for %s = %q, want %q", fn.QualifiedName, fn.Receiver, recv)
		}
		delete(want, fn.QualifiedName)
	}
	for n := range want {
		t.Errorf("missing function %q", n)
	}
}

func TestParseGoFile_HandlesEmptyInput(t *testing.T) {
	pf, err := ParseGoFile("empty.go", []byte{})
	if err != nil {
		t.Fatalf("parse empty: %v", err)
	}
	if len(pf.Functions) != 0 {
		t.Errorf("got %d functions on empty input", len(pf.Functions))
	}
}

func TestParseGoFile_TolerantOfSyntaxError(t *testing.T) {
	src := []byte(`package x
func ok() {}
this is not Go`)
	pf, err := ParseGoFile("bad.go", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(pf.Functions) == 0 {
		t.Errorf("expected at least the ok() function despite trailing garbage")
	}
}
