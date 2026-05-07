# Phase 0 smoke test

End-to-end fixture exercising the demo flow from `plan/00-tracer-zero.md` §4.

**Status:** scaffolded; populated by **P0.T49**.

## What this will test (when populated)

1. `graph-harness init` creates `.graph-harness/{overlay,policies}` and
   spawns the daemon.
2. A canonical `.gh` overlay (copied from `templates/overlay.example.gh`)
   declares a `CheckoutValidator` selector + `CheckoutValidation` flow.
3. `graph-harness selectors test CheckoutValidator` resolves to a single
   match against a fixture Go repo (`outcome: bound`, `confidence: ≥0.95`).
4. A pre-recorded diff that touches the matched function is fed to
   `graph-harness validate-diff`.
5. The pipeline emits exactly one `flow_unreviewed` finding (the only finding
   type Phase 0 produces) with the correct subject and severity.
6. Output is byte-identical across two consecutive runs at the same kernel
   sequence (idempotence check, plan §3 gate criterion 4).

## Layout (when populated)

```
tests/smoke/p0/
├── README.md         (this file)
├── run.sh            (orchestrator — invoked by `just smoke`)
├── fixture/          (a tiny self-contained Go module)
│   ├── go.mod
│   ├── main.go
│   └── internal/checkout/validator.go
├── overlay/checkout.gh
├── change.diff
└── expected_finding.json
```

The `just smoke` recipe in the root `justfile` checks for `run.sh` and is a
no-op until P0.T49 lands.
