// Package code_framework implements the code.framework layer: framework /
// app enrichment over code.core (Route, Handler, GraphQL resolver, Mutation,
// Event publisher/subscriber, Job, Queue consumer, Config key, Schema field,
// Migration, Fixture, Contract test).
//
// Phase 0 ships the layer registered, manifest valid, no extractors yet.
// **Working subset, not placeholder** — installs cleanly, accepts no events,
// returns empty results. Per-framework Go extractor plugins land in P2.
//
// SPEC: §2.3 (layer purpose), §6.15 (where framework sits in the pipeline).
//
// Phase 0 tasks: layer registration only (manifest in /manifests/code.framework.yaml).
package code_framework
