package kernel

import (
	"encoding/json"
	"slices"
	"time"
)

// EntityRef is a physical pointer (NOT durable identity).
// Per SPEC §4.1, durable identity is layer-canonical content-addressable
// keys; EntityRef is just a routing handle the kernel can use.
type EntityRef struct {
	Layer string `json:"layer"`
	Kind  string `json:"kind"`
	ID    string `json:"id"` // layer-local opaque ID
}

// FreshnessClass is the per-layer freshness state declared in the manifest.
// SPEC §2.3 lists per-layer enums; the kernel only cares about the
// blocking-risk order documented in SPEC §4.6.
type FreshnessClass string

// Freshness classes from SPEC §4.6.
const (
	FreshnessLive          FreshnessClass = "live"
	FreshnessCurrent       FreshnessClass = "current"
	FreshnessPossiblyStale FreshnessClass = "possibly_stale"
	FreshnessDirty         FreshnessClass = "dirty"
	FreshnessStale         FreshnessClass = "stale"
	FreshnessUnknown       FreshnessClass = "unknown"
	FreshnessUnresolved    FreshnessClass = "unresolved"
)

// blockingRiskOrder maps freshness to a strictly increasing risk score
// so callers can compute "is this safe to enforce on?" independently of
// the fold order. Higher = riskier. SPEC §4.6.
var blockingRiskOrder = map[FreshnessClass]int{
	FreshnessLive:          0,
	FreshnessCurrent:       1,
	FreshnessPossiblyStale: 2,
	FreshnessDirty:         3,
	FreshnessStale:         4,
	FreshnessUnknown:       5,
	FreshnessUnresolved:    6,
}

// BlockingRisk returns the risk score (lower is safer).
func BlockingRisk(f FreshnessClass) int {
	if v, ok := blockingRiskOrder[f]; ok {
		return v
	}
	return blockingRiskOrder[FreshnessUnknown]
}

// SourceClass tags where a fact came from (for provenance).
type SourceClass string

// Source classes for fact provenance (SPEC §4.4).
const (
	SourceExtractorTreeSitter SourceClass = "extractor:tree_sitter"
	SourceExtractorLSP        SourceClass = "extractor:lsp"
	SourceExtractorSCIP       SourceClass = "extractor:scip"
	SourceImporterGH          SourceClass = "importer:gh_filesystem"
	SourceImporterBenchFix    SourceClass = "importer:bench_fixture"
	SourceImporterBenchOracle SourceClass = "importer:bench_oracle"
	SourceUser                SourceClass = "user"
	SourceAgent               SourceClass = "agent"
	SourceLayerInternal       SourceClass = "layer_internal"
	SourceKernelReplay        SourceClass = "kernel_replay"
)

// Provenance records who said what, when, and how confident they are.
// SPEC §4.4. The fold (SPEC §4.5) computes a summary across constituents.
type Provenance struct {
	Confidence   float64        `json:"confidence"`
	Freshness    FreshnessClass `json:"freshness"`
	SourceClass  []SourceClass  `json:"source_class"`
	Inputs       []string       `json:"inputs"` // upstream event/entity IDs
	ProducedSeq  uint64         `json:"produced_seq"`
	Constituents []Provenance   `json:"constituents,omitempty"`
}

// Event is the immutable causal fact emitted by every layer.
// SPEC §4.2.
type Event struct {
	Seq        uint64          `json:"seq"`
	TS         time.Time       `json:"ts"`
	Layer      string          `json:"layer"`
	Kind       string          `json:"kind"`
	Subject    *EntityRef      `json:"subject,omitempty"`
	Payload    json.RawMessage `json:"payload"`
	Causes     []uint64        `json:"causes,omitempty"` // upstream seqs
	ProducedBy SourceClass     `json:"produced_by"`
	Tx         string          `json:"tx,omitempty"` // transaction ID for atomicity at subscribers
}

// LayerQuery is the request envelope passed to Facts.ReadCurrent / ReadAsOf.
// SPEC §6.4 leaves the shape to layers; the kernel only routes.
type LayerQuery struct {
	Capability string          `json:"capability"`
	Args       json.RawMessage `json:"args"`
}

// ResultEnvelope wraps a layer response with provenance + reproducibility.
// SPEC §4.7 read consistency: results carry the seq they were materialized at.
type ResultEnvelope struct {
	Data            json.RawMessage `json:"data"`
	ResolvedAtSeq   uint64          `json:"resolved_at_seq"`
	Provenance      Provenance      `json:"provenance"`
	NotReproducible bool            `json:"not_reproducible,omitempty"`
}

// SnapshotHandle is an opaque kernel-issued reference to a snapshot.
type SnapshotHandle struct {
	ID    string `json:"id"`
	Seq   uint64 `json:"seq"`
	Layer string `json:"layer"`
}

// EventFilter is parsed from the same Participle grammar as selectors.
// Phase 0 supports kind+layer equality only; richer expressions land in P3.
type EventFilter struct {
	Layers []string `json:"layers,omitempty"`
	Kinds  []string `json:"kinds,omitempty"`
}

// Matches reports whether an event satisfies the filter.
func (f EventFilter) Matches(e Event) bool {
	if len(f.Layers) > 0 && !slices.Contains(f.Layers, e.Layer) {
		return false
	}
	if len(f.Kinds) > 0 && !slices.Contains(f.Kinds, e.Kind) {
		return false
	}
	return true
}
