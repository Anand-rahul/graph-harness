# dsl-rejects-datalog-rules-with-deferred-message

## Why this spec exists

Phase 0 ships a deliberately small DSL surface (SPEC §11.2 + plan §P0.T15):
selector, flow, query, import, plus `match … return …` Cypher bodies. Datalog
`rule … :- …` declarations require the rule engine (`google/mangle` behind the
`RuleEngine` interface), which lands at **Phase 3** (`plan/03-rule-engine-and-repairs.md`).

Until then, accepting a `rule` declaration silently would mean either:

1. Storing the rule and silently failing to evaluate it later (worst — invisible
   broken state in `semantic.overlay`).
2. Throwing a confusing parse-tree error that says "invalid input near `:-`".

This spec asserts the third, correct behavior: the parser detects the rule
form pre-flight and rejects it with the exact phrase **"datalog rules deferred
to P3"**, which is greppable and points the reader at the right phase plan.

## Wave-graduation

When P3 lands, this spec stays green if the parser still accepts (and now
evaluates) `rule …` declarations *and* the assertion text changes — author
expectations move into `phase3/dsl/`, and this spec becomes a regression
guard against accidentally re-disabling rules.

## Reference

- SPEC §11.2 — Surface form
- SPEC §6.10 — Rule engine: Mangle behind a RuleEngine interface
- plan §P0.T15 — DSL parser scope
- plan/03-rule-engine-and-repairs.md — full rule-engine arrival
