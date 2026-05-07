package source_live

import (
	"strings"
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// TestTreeSitterSmoke verifies the cgo binding actually parses Go source.
// If this fails, every downstream P0.D task (P0.T18+) is blocked.
func TestTreeSitterSmoke(t *testing.T) {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(tree_sitter.NewLanguage(tree_sitter_go.Language())); err != nil {
		t.Fatalf("set language: %v", err)
	}
	src := []byte("package main\nfunc Hello() string { return \"hi\" }\n")
	tree := parser.Parse(src, nil)
	defer tree.Close()
	got := tree.RootNode().ToSexp()
	if !strings.Contains(got, "function_declaration") {
		t.Fatalf("expected function_declaration in s-expression, got: %s", got)
	}
}
