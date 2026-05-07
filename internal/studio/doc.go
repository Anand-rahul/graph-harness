// Package studio is the local-first web UI for the semantic layer.
//
// Phase 0 ships:
//   - HTTP server bound loopback-only with a per-session random token + strict
//     Origin check (SPEC §9.6 — these protections are non-negotiable).
//   - Embedded SPA shell via Go's embed.FS (TypeScript source compiled at
//     build time; deliberately thin in P0 — textarea + Prism syntax
//     highlighting; SolidJS / React decision deferred to P3).
//   - Two pages: .gh overlay file editor + selector live-preview against
//     code.core.
//
// The Studio writes overlay edits as canonical .gh files to disk, then the
// .gh watcher in semantic.overlay re-imports — Studio never bypasses the
// canonical event path.
//
// SPEC: §9.6 (Studio), §11.3 (authoring UX), §11.4 (live selector preview).
//
// Phase 0 tasks: P0.T42 (HTTP server + auth), P0.T43 (SPA editor + preview).
package studio
