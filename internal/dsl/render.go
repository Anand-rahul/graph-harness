package dsl

import (
	"fmt"
	"strings"
)

// Render turns a File AST back into canonical .gh source. Round-trip property:
// Parse → Render → Parse produces an identical AST. Used by the importer
// (semantic.overlay) to canonicalize on-disk files and by the test suite
// for round-trip property tests (P0.T16).
func Render(f *File) string {
	var b strings.Builder
	for i, d := range f.Decls {
		if i > 0 {
			b.WriteString("\n")
		}
		switch {
		case d.Import != nil:
			fmt.Fprintf(&b, "import %q as %s\n", d.Import.Path, d.Import.Alias)
		case d.Selector != nil:
			renderSelector(&b, d.Selector)
		case d.Flow != nil:
			renderFlow(&b, d.Flow)
		case d.Query != nil:
			fmt.Fprintf(&b, "query %q {\n%s\n}\n", d.Query.Name, strings.TrimSpace(d.Query.Body))
		case d.Rule != nil:
			// Should never reach here in P0 (validateP0Surface rejects),
			// but keep symmetric for future compatibility.
			fmt.Fprintf(&b, "rule %s(%s) :-%s.\n", d.Rule.Head, strings.TrimSpace(d.Rule.Args), strings.TrimSpace(d.Rule.Body))
		}
	}
	return b.String()
}

func renderSelector(b *strings.Builder, s *Selector) {
	fmt.Fprintf(b, "selector %s {\n", s.Name)
	if s.Unique {
		b.WriteString("  unique\n")
	}
	for _, a := range s.Anchors {
		fmt.Fprintf(b, "  anchor %s ", a.Kind)
		writeLit(b, a.Value)
		b.WriteString("\n")
	}
	b.WriteString("}\n")
}

func renderFlow(b *strings.Builder, f *Flow) {
	fmt.Fprintf(b, "flow %s {\n", f.Name)
	if f.Description != "" {
		fmt.Fprintf(b, "  description %q\n", f.Description)
	}
	if f.Scope != "" {
		fmt.Fprintf(b, "  scope %s\n", f.Scope)
	}
	if f.Risk != "" {
		fmt.Fprintf(b, "  risk %s\n", f.Risk)
	}
	for _, s := range f.Steps {
		fmt.Fprintf(b, "  step %s targets selector { ", s.Name)
		if s.Targets != nil && s.Targets.InlineSelector != nil {
			for i, a := range s.Targets.InlineSelector.Anchors {
				if i > 0 {
					b.WriteString("; ")
				}
				fmt.Fprintf(b, "%s ", a.Kind)
				writeLit(b, a.Value)
			}
		}
		b.WriteString(" }\n")
	}
	b.WriteString("}\n")
}

func writeLit(b *strings.Builder, l *Lit) {
	if l == nil {
		return
	}
	switch {
	case l.Str != nil:
		fmt.Fprintf(b, "%q", *l.Str)
	case l.Int != nil:
		fmt.Fprintf(b, "%d", *l.Int)
	case l.Float != nil:
		fmt.Fprintf(b, "%g", *l.Float)
	}
}
