// Package kernel implements the Graph Harness kernel: layer registry,
// dependency-DAG enforcement, the global event sequencer, snapshot semantics,
// and the cross-layer provenance fold.
//
// The kernel owns durable semantics; engines (parser, rule engine, storage
// backends) are replaceable behind interfaces (SPEC §6.8 — "kernel owns,
// engines provide").
//
// SPEC: §2 (architecture), §4 (data model), §5 (manifests), §6.13 (internal
// tables), §6.16 (event bus mechanics).
//
// Phase 0 tasks: P0.T05 (event log), P0.T06 (layer_state cursors),
// P0.T07 (registry + manifest validator), P0.T10 (provenance fold),
// P0.T11 (snapshot create/restore).
package kernel
