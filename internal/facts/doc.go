// Package facts defines the Facts Go interface (SPEC §6.4) — the universal,
// capability-gated storage adapter contract that every layer's backend
// implements. Cross-layer atomicity is the kernel's responsibility via the
// global sequencer; adapters are layer-local.
//
// The default v1 adapter is SQLite (modernc.org/sqlite, pure-Go) per SPEC
// §6.2–§6.3. Per-layer adapters may also embed in-memory caches, filesystem
// indexes, or — when benchmark evidence demands — DuckDB / OxiGraph / Cozo.
//
// SPEC: §6.1 (Facts interface above every backend), §6.4 (adapter contract),
// §6.5 (bitemporality), §6.13 (kernel-owned tables).
//
// Phase 0 tasks: P0.T08 (Facts interface), P0.T09 (SQLite adapter).
package facts
