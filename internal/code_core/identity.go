// Package code_core implements the code.core layer: content-addressable
// canonical IDs (SPEC §6.12) plus the SQLite-backed adjacency store for
// `calls` and `references` relations.
package code_core

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// EntityKind enumerates the kinds materialized into code.core in P0.
// Single-source (tree-sitter) per plan §P0.T21.
type EntityKind string

// Phase 0 entity kinds materialized by code.core.
const (
	KindFile     EntityKind = "File"
	KindFunction EntityKind = "Function"
	KindMethod   EntityKind = "Method"
	KindTypeDecl EntityKind = "TypeDecl"
)

// Entity is a materialized code.core entity with its content-addressable ID.
type Entity struct {
	ID            string
	Kind          EntityKind
	LanguageID    string
	QualifiedName string
	Receiver      string // methods only
	Path          string // files only
	BodyHash      string // functions only
}

// FileID = sha256(workspace_relative_path) per SPEC §6.12.
func FileID(workspaceRelPath string) string {
	return sha256Hex(workspaceRelPath)
}

// FunctionID = sha256(language_id || qualified_name || normalized_signature).
func FunctionID(languageID, qualifiedName, normalizedSig string) string {
	return sha256Hex(languageID + "\x00" + qualifiedName + "\x00" + normalizedSig)
}

// MethodID = sha256(language_id || receiver_qualified_name || method_name || normalized_signature).
func MethodID(languageID, receiverQN, methodName, normalizedSig string) string {
	return sha256Hex(languageID + "\x00" + receiverQN + "\x00" + methodName + "\x00" + normalizedSig)
}

// TypeDeclID = sha256(language_id || qualified_name).
func TypeDeclID(languageID, qualifiedName string) string {
	return sha256Hex(languageID + "\x00" + qualifiedName)
}

// NormalizeGoSignature collapses whitespace and trims so formatting-only
// edits do not change a function's identity. Per SPEC §6.12 the normalizer
// is per-language and pluggable; the P0 implementation is intentionally
// conservative — full type-aware normalization (param-name elision, generic
// param canonicalization) requires LSP/SCIP-grade analysis that lands in P1.
//
// The body_hash anchor in selectors handles formatting equivalence at the
// semantic level until then.
func NormalizeGoSignature(sig string) string {
	s := whitespaceRE.ReplaceAllString(sig, " ")
	s = strings.TrimSpace(s)
	// Trim spaces directly inside parentheses and brackets so
	// `( a int )` and `(a int)` produce the same canonical form.
	s = openParenSpaceRE.ReplaceAllString(s, "$1")
	s = closeParenSpaceRE.ReplaceAllString(s, "$1")
	return s
}

var (
	whitespaceRE      = regexp.MustCompile(`\s+`)
	openParenSpaceRE  = regexp.MustCompile(`([\(\[])\s+`)
	closeParenSpaceRE = regexp.MustCompile(`\s+([\)\]])`)
)

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
