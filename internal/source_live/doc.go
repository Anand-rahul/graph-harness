// Package source_live implements the source.live layer: the raw extractor
// fact stream that feeds code.core. v1 unifies three independent fact
// sources — LSP (live), SCIP (indexed), tree-sitter (structural) — via
// content-addressable keys and conflict-as-event semantics (SPEC §6.11).
//
// Phase 0 ships tree-sitter only (Go grammar via tree-sitter/go-tree-sitter,
// the only approved cgo dependency for v1 per SPEC §6.17). LSP + SCIP join
// in P1 along with TS + Python.
//
// File watching uses fsnotify (SPEC §6.16, plan §P0.T19): on change, re-parse
// the touched file and emit FileChanged + FileParsed events into the layer.
//
// SPEC: §2.3 (layer purpose + freshness states), §6.11 (three-input model),
// §6.15 (code-fact pipeline summary).
//
// Phase 0 tasks: P0.T18 (tree-sitter integration), P0.T19 (fsnotify watcher),
// P0.T20 (Facts adapter + manifest).
package source_live
