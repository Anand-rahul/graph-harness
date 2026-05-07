# studio-binds-loopback-and-mints-token

## Why this spec exists

Studio is a local-first web UI (SPEC §9.6) that exposes overlay editing,
selector resolution, finding inspection, and (later) review-queue actions
over HTTP. Three constraints are **non-negotiable** and apply from day one:

1. **Loopback-only bind.** Studio listens on `127.0.0.1` (or `[::1]`), never
   on a routable interface. A misconfigured `0.0.0.0` bind would expose the
   workspace to anyone on the LAN — including the overlay editor that writes
   to disk.
2. **Per-session token.** Each Studio invocation mints a fresh random token
   (16 bytes of crypto/rand → 32 hex chars). The token is included in the
   printed URL and required on every request. Tokens never persist across
   invocations.
3. **Strict Origin check.** When an `Origin` header is present, it must
   start with `http://127.0.0.1` or `http://localhost`. A browser request
   from `https://attacker.example` fails with 403 even if the token leaks.

This spec verifies (1) and (2) directly via `--oneshot` mode (the server
binds, prints the URL, exits cleanly). The Origin check is exercised in
P4 surface-specs once richer Studio routes land.

## Why CI uses --oneshot

The bare `graph-harness studio` blocks indefinitely (`http.Server.Serve`).
gotit specs are linear and bounded; `--oneshot` lets us assert the bind +
token-mint behavior without a long-running process. Future Studio specs
(P4) will exercise actual HTTP roundtrips against a `--port=N` server in
a setup step.

## Reference

- SPEC §9.6 — Graph Harness Studio
- plan §P0.T42 — Studio HTTP server + auth
- plan/04-human-loop-closure.md §7 — full Studio specs at P4
