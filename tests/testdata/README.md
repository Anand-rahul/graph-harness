# tests/testdata

Test fixtures consumed by `_test.go` files. Anything under this directory is
treated as data-only by Go's tooling (the leading `testdata/` segment is
ignored by `go build`).

Conventions:

- One subdirectory per fixture domain (e.g. `dsl/`, `kernel/`, `code_core/`).
- Goldens live next to the test that owns them.
- Linters (`golangci-lint`) skip this tree; see `.golangci.yml`.
- Large binary fixtures (>200 KB) belong in a generator script, not the tree.
