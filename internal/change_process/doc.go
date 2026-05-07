// Package change_process implements the change.process layer: the per-change
// validation pipeline that produces ValidationFinding and RepairInstruction
// events. The pipeline is twelve stages (SPEC §8.1), idempotent at a fixed
// kernel sequence.
//
// Phase 0 stage status (plan §1):
//
//  1. parse diff                         FULL
//  2. map hunks → code.core entities     FULL
//  3. bounded refresh of source.live     WORKING SUBSET
//     3b. pin validation_seq + dirty/stale  WORKING SUBSET
//  4. compute touched_set                FULL
//  5. compute impacted_set (BFS only)    FULL  (Mangle integration → P3)
//  6. selector resolution + flow match   FULL
//  7. invariant check                    PASS-THROUGH (P3)
//  8. control evaluation                 PASS-THROUGH (P3)
//  9. agent / skill hooks                PASS-THROUGH
//  10. emit ValidationFinding events     FULL  (one finding type only)
//  11. RepairInstruction synthesis       PASS-THROUGH (P3)
//  12. emit ValidateDiffResult summary   FULL
//
// Pass-through stages are working subsets, not placeholders: they advance
// the pipeline correctly even when emitting empty output. Same idempotence
// guarantee applies.
//
// SPEC: §8 (validation pipeline), §6.14 (traversal — used in stage 5).
//
// Phase 0 tasks: P0.T28–P0.T36.
package change_process
