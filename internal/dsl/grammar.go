// Package dsl is the single Participle v2 parser for the .gh surface form.
//
// Phase 0 scope per plan §P0.T15: selector, flow, query, import declarations
// plus Cypher `match … return …` query bodies. Datalog `rule … :- …` is
// rejected with a clear error message until P0.C lands.
package dsl

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// File is a parsed .gh source file. Authors mix declarations freely.
type File struct {
	Decls []*Decl `@@*`
}

// Decl is one top-level declaration. Exactly one branch is non-nil.
type Decl struct {
	Import   *Import   `  @@`
	Selector *Selector `| @@`
	Flow     *Flow     `| @@`
	Query    *Query    `| @@`
	Rule     *Rule     `| @@`
}

// Import lifts another .gh file into the current scope, namespaced.
//
//	import "selectors/payments.gh" as p
type Import struct {
	Path  string `"import" @String`
	Alias string `"as" @Ident`
}

// Selector declares a named multi-anchor matcher.
type Selector struct {
	Name    string    `"selector" @Ident "{"`
	Unique  bool      `( @"unique"`
	Anchors []*Anchor `| @@ )*`
	End     struct{}  `"}"`
}

// Anchor is one entry in the multi-anchor ladder.
type Anchor struct {
	Kind  string `"anchor" @Ident`
	Value *Lit   `( @@ )?`
}

// Lit is a string, integer, or float literal.
type Lit struct {
	Str   *string  `  @String`
	Int   *int     `| @Int`
	Float *float64 `| @Float`
}

// Flow declares a multi-step semantic flow.
type Flow struct {
	Name        string      `"flow" @Ident "{"`
	Description string      `( "description" @String )?`
	Scope       string      `( "scope" @Ident )?`
	Risk        string      `( "risk" @Ident )?`
	Steps       []*FlowStep `( @@ )*`
	End         struct{}    `"}"`
}

// FlowStep is one named step in a flow.
type FlowStep struct {
	Name    string  `"step" @Ident`
	Targets *Target `"targets" @@`
}

// Target is the resolution target for a step. v0 supports inline-selector form.
type Target struct {
	InlineSelector *InlineSelector `"selector" @@`
}

// InlineSelector is a one-off selector with anchors literal-encoded.
type InlineSelector struct {
	Anchors []*InlineAnchor `"{" @@+ "}"`
}

// InlineAnchor is a `<kind> <value>` inside an inline selector body.
type InlineAnchor struct {
	Kind  string `@Ident`
	Value *Lit   `@@`
}

// Query is a named DSL query. v0 supports Cypher match/return only.
type Query struct {
	Name string   `"query" @String "{"`
	Body string   `@( ~"}" )*`
	End  struct{} `"}"`
}

// Rule is the Datalog form. P0 rejects these in semantic validation; the
// grammar accepts them so we can produce a clear "deferred to P3" error.
type Rule struct {
	Head string `"rule" @Ident`
	Args string `"(" @( ~")" )* ")"`
	Body string `":-" @( ~"." )* "."`
}

var ghLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Comment", Pattern: `(?:\/\/[^\n]*|\/\*[\s\S]*?\*\/)`},
	{Name: "Float", Pattern: `[-+]?\d+\.\d+`},
	{Name: "Int", Pattern: `[-+]?\d+`},
	{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_\.]*`},
	{Name: "String", Pattern: `"(\\"|[^"])*"`},
	{Name: "Punct", Pattern: `[\{\}\(\)\[\],:=]`},
	{Name: "whitespace", Pattern: `[ \t\r\n]+`},
})

var fileParser = participle.MustBuild[File](
	participle.Lexer(ghLexer),
	participle.Elide("Comment"),
	participle.Unquote("String"),
	participle.UseLookahead(2),
)

// Parse parses the .gh source from r into a File AST.
func Parse(r io.Reader, name string) (*File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseString(name, string(data))
}

// ParseString parses .gh source from a string.
func ParseString(name, source string) (*File, error) {
	// Pre-flight: Datalog rules are deferred to P3 per plan §P0.T15. The
	// lexer does not handle `:-` so a Datalog body produces a confusing
	// "invalid input" error. Detect the pattern first and surface the
	// project-specific message.
	if hasDatalogRule(source) {
		return nil, fmt.Errorf("datalog rules deferred to P3; v0 supports selector/flow/query/import only")
	}
	f, err := fileParser.ParseString(name, source)
	if err != nil {
		return nil, err
	}
	if err := validateP0Surface(f); err != nil {
		return nil, err
	}
	return f, nil
}

// hasDatalogRule scans source for the `rule <ident>(...) :- ...` pattern.
// Conservative: any `:-` outside a string literal triggers rejection.
func hasDatalogRule(src string) bool {
	in := src
	// Strip string contents to avoid false positives.
	for {
		i := strings.IndexByte(in, '"')
		if i < 0 {
			break
		}
		j := strings.IndexByte(in[i+1:], '"')
		if j < 0 {
			break
		}
		in = in[:i] + in[i+j+2:]
	}
	return strings.Contains(in, ":-")
}

// validateP0Surface enforces SPEC §11.2 + plan §P0.T15: Datalog rules are
// deferred to P3 with a clear message. Callers see a single, helpful error
// rather than discovering it during evaluation later.
func validateP0Surface(f *File) error {
	for _, d := range f.Decls {
		if d.Rule != nil {
			return fmt.Errorf("datalog rules deferred to P3 (rule %s); v0 supports selector/flow/query/import only", d.Rule.Head)
		}
	}
	return nil
}
