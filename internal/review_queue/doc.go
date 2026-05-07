// Package review_queue implements the review.queue layer: the proposal
// state machine that mediates writes from non-trusted sources (agents,
// importers) into authoritative layers.
//
// Full v1 state set (SPEC §10.1):
//
//	pending → accepted / accepted_with_edits / rejected / deferred /
//	          needs_evidence / resolved (+ edit_requested)
//
// Phase 0 ships a working subset:
//
//	submitted | pending_review | accepted | rejected | promoted
//
// Auto-validation is JSON-AST schema check only. Promotion mode is
// `human_only` for every layer in P0. Conflict-as-event scaffolded but
// only selector-overlap detected; invariant-contradiction lands in P3.
//
// Trust-policy enforcement: writes from non-`direct_writes_from` sources
// auto-convert to Proposals (SPEC §5.4 + §10.2).
//
// SPEC: §10 (review queue lifecycle), §5.4 (trust policy).
//
// Phase 0 tasks: P0.T37 (manifest), P0.T38 (state machine + trust enforcement),
// P0.T39 (selector-overlap conflict detection), P0.T40 (CLI).
package review_queue
