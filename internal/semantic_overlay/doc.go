// Package semantic_overlay implements the semantic.overlay layer: the durable,
// human-authored semantics of the system (Flow, Invariant, Control, Skill,
// Selector, Runbook, Risk label, Boundary, Ownership hint).
//
// Source-of-truth is filesystem-canonical .gh files under
// .graph-harness/overlay/**; SQLite is the index. The importer reads .gh,
// the DSL parser produces a JSON AST, and the layer emits OverlayItemImported
// events. This makes the overlay git-tracked, PR-reviewable, and durable.
//
// Selector resolution is the central operation. v1 anchor families are closed
// (SPEC §3.1) with one `graph_query { dsl }` escape hatch. Resolution returns
// a structured envelope (SPEC §3.2) with five outcomes: bound, reanchored,
// ambiguous, unresolved, superseded.
//
// Phase 0 ships single-anchor resolution with outcomes {bound, unresolved}
// only. Multi-anchor + full 5-outcome lands in P3 alongside drift events.
//
// SPEC: §2.3, §3 (selectors), §11 (.gh DSL), §4.8 (event log + overlay
// canonicality).
//
// Phase 0 tasks: P0.T24 (.gh watcher + import events),
// P0.T25 (Facts adapter + import/materialize cycle),
// P0.T26 (single-anchor selector resolution + cache invalidation).
package semantic_overlay
