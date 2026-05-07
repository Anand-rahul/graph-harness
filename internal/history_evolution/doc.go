// Package history_evolution implements the history.evolution layer: derived
// commit-replay artifacts (Commit, EntityRevision, RelationshipRevision, Move,
// Rename, Split, Merge, CoChange, HistoricalFlowChange, StalenessSignal).
//
// Phase 0 ships the layer registered, manifest valid, no commit ingestion
// yet. Working subset returning empty results. Bitemporality follows the
// XTDB pattern (SPEC §6.5): `valid_from/valid_to` (domain time) +
// `tx_from/tx_to` (kernel seq).
//
// Optional analytics adapter is DuckDB (deferred to post-P0; cgo activation
// requires a fresh deviation review per SPEC §6.17).
//
// SPEC: §2.3, §6.5 (bitemporality), §6.6 (sync mode = pull_replica).
//
// Phase 0 tasks: layer registration only (manifest in /manifests/history.evolution.yaml).
// Commit ingestion lands in P5.
package history_evolution
