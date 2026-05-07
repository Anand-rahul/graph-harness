// Package tui implements the terminal cockpit (charmbracelet/bubbletea).
//
// Phase 0 ships a read-only skeleton: two views — workspace status and
// finding list — and a clean Ctrl-C exit. Full lifecycle (review queue
// accept/reject/defer, finding suppress, plan adherence overlay) lands in P4.
//
// SPEC: §9.5 (TUI), §11.3 (authoring UX — TUI is read-mostly).
//
// Phase 0 tasks: P0.T41 (skeleton + two views).
package tui
