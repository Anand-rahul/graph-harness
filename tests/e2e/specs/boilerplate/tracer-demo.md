# tracer-demo

The capstone spec for the `boilerplate` wave. It exercises the
`plan/00-tracer-zero.md` §4 demo end-to-end: every architectural component
(kernel, DSL parser, source.live, code.core, semantic.overlay, change.process,
CLI) is touched by a single linear scenario.

**Wave:** boilerplate
**Tags:** boilerplate, demo, capstone, p0t49
**Spec:** `tracer-demo.yaml`

## Why this spec is the wave gate

Phase 0's whole discipline is "stable, demonstrable end-to-end product with
every component touched in some form" (plan §1). If this spec passes, Phase 0
holds together; if it regresses, the spine is broken — period.

The smoke test under `tests/smoke/p0/run.sh` is a thin wrapper that runs
*only this spec*, so `just smoke` and the capstone stay byte-identical.

## Scenario

1. Helper `go-module-with-checkout-validator` creates a tiny Go module with a
   `CheckoutValidator.Validate` method, then pre-stages `demo.checkout.gh`
   (selector + flow) and `demo.diff` (touches `Validate` without acknowledging
   the flow).
2. `graph-harness init` scaffolds `.graph-harness/`.
3. The overlay is dropped into `.graph-harness/overlay/checkout.gh`.
4. `graph-harness selectors test CheckoutValidator` resolves to one match
   (`outcome: bound`, confidence ≥ 0.9, `via_anchor: qualified_name`).
5. `graph-harness flows list` reports the flow.
6. `graph-harness validate-diff --diff demo.diff` runs the 12-stage pipeline
   and emits exactly one `flow_unreviewed` finding subject to
   `checkout.CheckoutValidator.Validate`.
7. The same command with `--json` confirms the SPEC §8.2 finding shape.
8. Re-running `validate-diff` produces byte-identical output (idempotence).
9. An empty diff yields zero findings.

## Why the empty `repair: {}`

P0's only finding kind is `flow_unreviewed`; structured `RepairInstruction`
synthesis lives in P3. The shape contract carries `repair` as a required
field but accepts an empty object until then. Specs under `phase3/repairs/`
will replace empty-payload assertions with full-shape ones once the rule
engine lands.

## Reference

- `plan/00-tracer-zero.md` §4 — the demo this spec implements
- `plan/00-tracer-zero.md` §3 — gate criteria (smoke test, idempotence)
- SPEC §8.1 — twelve-stage pipeline
- SPEC §8.2 — ValidationFinding shape
- SPEC §3.2 — selector resolution outcomes
