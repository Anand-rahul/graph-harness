// Package dsl is the single Participle v2 (alecthomas/participle) parser for
// the .gh surface form: selectors, queries, flows, invariants, controls,
// skills, rules, imports. The parser produces a canonical JSON AST that the
// kernel imports; rendering is bidirectional (.gh ↔ AST) and deterministic.
//
// Selector DSL and query DSL share one parser and one canonical AST model
// (SPEC §6.9). The Participle grammar is also used to parse event-bus filter
// expressions (SPEC §6.16).
//
// Phase 0 scope is intentionally thin: selector / flow / query / import +
// Cypher `match … return` only. Datalog `rule … :-` lands in P3 with the
// rule engine.
//
// SPEC: §3 (selectors), §6.9 (parser choice), §7 (query language),
// §11 (DSL grammar surface).
//
// Phase 0 tasks: P0.T15 (Participle grammar + JSON AST + render),
// P0.T16 (round-trip property tests).
package dsl
