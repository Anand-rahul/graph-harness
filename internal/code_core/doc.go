// Package code_core implements the code.core layer: the normalized,
// language-level code graph (Repo, Workspace, File, Module, Package, Symbol,
// Function, Class, Method, Interface, Import/Export, Call, Reference,
// Definition, Test, GeneratedArtifact).
//
// Identity is content-addressable per SPEC §6.12:
//
//	File:        sha256(workspace_relative_path)
//	Module/Pkg:  sha256(language_id || qualified_name)
//	Function:    sha256(language_id || qualified_name || normalized_signature)
//	Method:      sha256(language_id || receiver_qualified_name || method_name || normalized_signature)
//	Class/Iface: sha256(language_id || qualified_name)
//	Symbol:      sha256(language_id || qualified_name || kind_tag)
//	Anonymous:   sha256(parent_id || kind_tag || ordinal_within_parent)
//
// Three-source unification rule is conflict-as-event: when LSP, SCIP, and
// tree-sitter agree, the entity is one with merged provenance; when they
// disagree, both are recorded and a code.core.SymbolDisambiguation event is
// emitted (never silently merged). Renames produce new IDs + supersession.
//
// Graph traversal in v1 is adjacency tables in SQLite + planner BFS/DFS
// operators + Mangle for recursion (SPEC §6.14). No property-graph database.
//
// SPEC: §6.11 (three-input model), §6.12 (identity), §6.13 (kernel tables),
// §6.14 (traversal), §6.15 (pipeline summary).
//
// Phase 0 tasks: P0.T21 (identity scheme + Go signature normalizer),
// P0.T22 (Facts adapter + tree-sitter event consumer),
// P0.T23 (adjacency table + planner BFS).
package code_core
