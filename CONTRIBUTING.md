# Contributing to graph-harness

Thanks for your interest. graph-harness is in **early development** — APIs, the
`.gh` format, and the CLI surface are likely to change before v0.1. Issues,
discussions, and PRs are all welcome; expect rough edges.

## Dev setup

- Go 1.22+
- [`just`](https://github.com/casey/just) for the task runner
- A C toolchain (`cc` or `gcc`) — required for tree-sitter cgo bindings

Clone, then:

```sh
just build       # compiles bin/graph-harness
just test        # unit tests with -race
just lint        # golangci-lint v2
just fmt         # gofumpt
just e2e         # end-to-end specs (see tests/e2e/README.md)
```

`just --list` shows every recipe.

## End-to-end tests

The E2E suite uses [gotit](https://github.com/shivamstaq/gotit) to drive YAML
specs against the built binary. See [`tests/e2e/README.md`](tests/e2e/README.md)
for layout, how to run a single wave, and how to add a spec.

## Style

- Code is formatted with `gofumpt` and linted with `golangci-lint v2` — both
  wired in `justfile`. CI runs `just fmt-check` and `just lint`.
- Keep changes focused. One concern per PR.
- Public-facing names follow standard Go conventions; package layout follows
  the existing `internal/<area>/` split.

## Proposing a change

- For non-trivial changes (new commands, format extensions, new layer
  semantics), open an issue first so we can align on shape before you write
  code.
- Small fixes (typos, doc updates, obvious bugs) can go straight to a PR.
- No CLA, no DCO sign-off requirement at this stage.

## Where to start

The README's [Contributing](README.md#contributing) section lists the current
call-outs — language extractors, framework adapters, harness templates,
importers from existing rule formats, and per-runtime bootstrap recipes for
`SKILL.md`. If something there matches your interest, file an issue saying so
and we'll scope it together.
