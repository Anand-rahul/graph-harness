package code_core

import "testing"

func TestNormalizeGoSignature_StableUnderFormatting(t *testing.T) {
	a := NormalizeGoSignature("(  a int,  b   string ) error")
	b := NormalizeGoSignature("(a int, b string) error")
	if a != b {
		t.Errorf("formatting changed signature: %q vs %q", a, b)
	}
}

func TestFunctionID_DifferentForDifferentLanguages(t *testing.T) {
	go1 := FunctionID("go", "checkout.Validate", "(any) error")
	ts1 := FunctionID("ts", "checkout.Validate", "(any) error")
	if go1 == ts1 {
		t.Errorf("language_id should distinguish identifiers across languages")
	}
}

func TestFunctionID_StableForSameInputs(t *testing.T) {
	a := FunctionID("go", "x.Y", "() error")
	b := FunctionID("go", "x.Y", "() error")
	if a != b {
		t.Errorf("same inputs produced different IDs: %s vs %s", a, b)
	}
}

func TestMethodID_IncludesReceiver(t *testing.T) {
	a := MethodID("go", "checkout.Validator", "Validate", "() error")
	b := MethodID("go", "checkout.OtherValidator", "Validate", "() error")
	if a == b {
		t.Errorf("different receivers must produce different IDs")
	}
}
