# End-to-end tests

End-to-end coverage for the `graph-harness` CLI. Specs are YAML files
executed by [gotit](https://github.com/shivamstaq/gotit), which builds the
binary and drives it as a subprocess against an isolated workspace per spec.

## Run

```sh
just e2e                       # all specs, headless
just e2e-wave boilerplate      # one wave (sub-tree of specs/)
just gotit                     # interactive TUI
just gotit-run boilerplate/cli # headless via the gotit binary, filtered
```

Or directly with `go test`:

```sh
go test ./tests/e2e/...
go test ./tests/e2e/ -run 'TestE2E/boilerplate/cli/'
```

The suite builds `./cmd/graph-harness` once per `go test` invocation and
runs specs in parallel with per-spec `HOME`, git repo, and `PATH`.

## Layout

```
tests/e2e/
├── runner_test.go      # single TestE2E entry point
├── gotit.yaml          # runner config (binary name, paths, env prefix)
├── features.yaml       # feature-flag allowlist for gated specs
├── specs/              # YAML specs grouped by sub-directory
├── helpers/            # Go: repo builders, requirement checks, assertions
└── results/            # last-run artifacts (gitignored)
```

## Adding a spec

1. Drop a `<name>.yaml` under the appropriate sub-directory of `specs/`.
   Names are kebab-case verbs (`init-rejects-double-init.yaml`). Existing
   specs in `specs/boilerplate/` are the reference for shape and tags.
2. If the spec needs a non-trivial repo state, add a builder in
   `helpers/repos.go` and register it in `runner_test.go` under
   `RepoHelpers`. Otherwise reuse `go-module-empty`.
3. If the spec depends on an unshipped feature, gate it with
   `requires: [feature:<flag>]` and add the flag to `features.yaml`
   (default `false`). Flip to `true` in the PR that lands the feature.
4. An optional `<name>.md` sidecar next to the YAML is the place for the
   *why* — useful for edge-case specs whose intent is not obvious.

## Custom assertions

Project-specific assertion types are prefixed `x-` and implemented in
`helpers/assertions.go` (e.g. `x-event-log-monotonic`,
`x-finding-shape-valid`). Register them in `runner_test.go` under
`Assertions`.

## What this suite does *not* cover

- Package-level invariants and algorithm correctness — those live next to
  the implementation as `*_test.go`.
- Property and fuzz tests — likewise in-package, using `testing/quick` or
  `gopter`.
