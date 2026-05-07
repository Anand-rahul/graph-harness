# Graph Harness — VS Code extension

**Status:** placeholder; populated by **P0.T44**.

This will be a TypeScript skeleton with:

- code lens on `.gh` declarations (live match count from `selectors test`
  over JSON-RPC),
- Problems pane integration for `change.process` findings,
- syntax highlighting via TextMate grammar (LSP-for-DSL deferred to P2+).

Build pipeline: `vsce package`. Sideloaded in P0; marketplace publishing
deferred to P5/P6 (plan §5 risks table).

The extension talks to the same daemon socket / named pipe as the CLI; there
is no second protocol (SPEC §9.4).
