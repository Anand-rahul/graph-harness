// Package jsonrpc hosts the daemon's JSON-RPC 2.0 server (sourcegraph/jsonrpc2)
// over a Unix domain socket on Linux/macOS and a named pipe on Windows. Method
// dispatch is capability-gated: only methods declared by an installed layer's
// manifest are reachable.
//
// All consumer surfaces — CLI, TUI, Studio, VS Code, MCP adapter — speak the
// same JSON-RPC surface to the daemon. There is no second protocol.
//
// SPEC: §6.16 (event bus mechanics — filter expressions share the DSL parser),
// §9.1 (daemon lifecycle), §9.9 (consumer surface summary).
//
// Phase 0 tasks: P0.T13 (server registration + method dispatch + capability
// gating).
package jsonrpc
