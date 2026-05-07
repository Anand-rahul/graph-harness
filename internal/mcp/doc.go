// Package mcp implements the MCP (Model Context Protocol) adapter that
// exposes Graph Harness capabilities to coding agents over the official
// modelcontextprotocol/go-sdk. The adapter is a curated, opinionated tool
// surface — not a thin pass-through (SPEC §9.3).
//
// Phase 0 v0 tools (plan §P0.T45):
//
//	explore        — bounded graph navigation around a selector
//	validate_diff  — run change.process on a candidate diff
//	get_context    — fetch flow / invariant / control evidence
//	query          — execute a DSL query
//
// Resources, prompts, and notifications are scaffolded with one example each.
// Stdio transport in P0 (HTTP/SSE deferred). All tool calls run trust-policy
// enforcement: writes from agents convert to Proposals via the review queue.
//
// SPEC: §9.3 (MCP adapter), §10.2 (promotion modes — agents are non-trusted
// by default).
//
// Phase 0 tasks: P0.T45.
package mcp
