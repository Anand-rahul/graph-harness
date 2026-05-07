// Package bench implements the bench.scenarios and bench.oracle layers and
// the `graph-harness bench` runner. The two layers are independent: per
// SPEC §12 + plan §P0.T46, bench.oracle has zero kernel write paths into
// the rest of the system, so it can never accidentally contaminate
// production state.
//
// Phase 0 ships:
//   - Both layers registered with valid manifests.
//   - One scenario fixture: scenario 1 (auth-sensitive edit, Go) — repo
//     snapshot at known seq + target diff + oracle ground truth.
//   - One regime: `mature` only.
//   - Runner that loads a scenario, runs the change.process pipeline at
//     snapshot seq, and scores the detection axis only against the oracle.
//     Other three axes (repair quality, repair execution, final clean-patch)
//     report N/A in P0 — populated alongside repairs in P3.
//
// The full four-axis scoring + ship-bar gate lands in P6.
//
// SPEC: §12 (benchmark methodology).
//
// Phase 0 tasks: P0.T46 (manifests), P0.T47 (scenario 1 fixture),
// P0.T48 (runner).
package bench
