// Package daemon owns the workspace daemon lifecycle: watchman-style
// lazy-spawn on first CLI invocation, pidfile + socket management under the
// platform runtime directory, and idle-timeout (default 1h, configurable).
//
// Platform paths:
//
//	Linux:   $XDG_RUNTIME_DIR/graph-harness/<workspace-hash>.sock
//	macOS:   ~/Library/Application Support/graph-harness/<workspace-hash>.sock
//	Windows: \\.\pipe\graph-harness-<workspace-hash>
//
// One workspace = one daemon = one shared graph cache. CI integration works
// through the same daemon path; a `--no-daemon` mode runs single-shot.
//
// SPEC: §9.1 (lazy-spawn), §9.7 (CI integration).
//
// Phase 0 tasks: P0.T12 (lifecycle, pidfile, socket; cross-platform).
package daemon
