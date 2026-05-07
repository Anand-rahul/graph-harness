// Package version exposes build metadata injected via -ldflags at link time.
//
// Defaults are sentinel values that indicate an unstamped build (e.g.
// `go run` or `go build` without the recommended -ldflags). The justfile
// `build` recipe sets all three.
package version

var (
	// Version is the semver-ish project version (e.g. "0.1.0-dev").
	Version = "dev"

	// Commit is the short git commit SHA the binary was built from.
	Commit = "none"

	// Date is the RFC3339 build timestamp.
	Date = "unknown"
)

// String returns a human-readable summary suitable for `--version` output.
func String() string {
	return Version + " (" + Commit + ", " + Date + ")"
}
