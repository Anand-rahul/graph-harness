package kernel

import (
	"slices"
	"testing"
)

func TestFoldProvenance_MinWorstUnionMax(t *testing.T) {
	in := []Provenance{
		{
			Confidence:  0.95,
			Freshness:   FreshnessLive,
			SourceClass: []SourceClass{SourceExtractorTreeSitter},
			Inputs:      []string{"e1"},
			ProducedSeq: 10,
		},
		{
			Confidence:  0.80,           // becomes the min
			Freshness:   FreshnessDirty, // becomes the worst
			SourceClass: []SourceClass{SourceExtractorLSP},
			Inputs:      []string{"e2"},
			ProducedSeq: 25, // becomes the max
		},
	}
	out := FoldProvenance(in)
	if out.Confidence != 0.80 {
		t.Errorf("confidence = %v, want 0.80", out.Confidence)
	}
	if out.Freshness != FreshnessDirty {
		t.Errorf("freshness = %s, want %s", out.Freshness, FreshnessDirty)
	}
	if out.ProducedSeq != 25 {
		t.Errorf("produced_seq = %d, want 25", out.ProducedSeq)
	}
	wantSrc := []SourceClass{SourceExtractorLSP, SourceExtractorTreeSitter}
	if !slices.Equal(out.SourceClass, wantSrc) {
		t.Errorf("source_class = %v, want %v", out.SourceClass, wantSrc)
	}
	if !slices.Equal(out.Inputs, []string{"e1", "e2"}) {
		t.Errorf("inputs = %v, want [e1 e2]", out.Inputs)
	}
	if len(out.Constituents) != 2 {
		t.Errorf("constituents preserved = %d, want 2", len(out.Constituents))
	}
}

func TestFoldProvenance_BlockingRiskIsSeparateFromFold(t *testing.T) {
	// Sanity check the documented split between fold order and blocking-risk
	// order (SPEC §4.6). For freshness specifically they coincide, but the
	// API distinguishes them so future code can diverge cleanly.
	if BlockingRisk(FreshnessLive) >= BlockingRisk(FreshnessUnresolved) {
		t.Errorf("blocking-risk ordering inverted: live=%d unresolved=%d",
			BlockingRisk(FreshnessLive), BlockingRisk(FreshnessUnresolved))
	}
}

func TestFoldProvenance_EmptyInput(t *testing.T) {
	out := FoldProvenance(nil)
	if out.Confidence != 0 || out.ProducedSeq != 0 || out.Freshness != "" {
		t.Errorf("empty fold = %+v", out)
	}
}

func TestFoldProvenance_DeterministicOutput(t *testing.T) {
	in := []Provenance{
		{Confidence: 0.9, Freshness: FreshnessCurrent, SourceClass: []SourceClass{SourceExtractorSCIP, SourceExtractorTreeSitter}, Inputs: []string{"a", "b"}, ProducedSeq: 5},
		{Confidence: 0.7, Freshness: FreshnessStale, SourceClass: []SourceClass{SourceExtractorLSP, SourceExtractorTreeSitter}, Inputs: []string{"b", "c"}, ProducedSeq: 7},
	}
	a := FoldProvenance(in)
	b := FoldProvenance(in)
	if !slices.Equal(a.SourceClass, b.SourceClass) || !slices.Equal(a.Inputs, b.Inputs) {
		t.Errorf("non-deterministic fold: %+v vs %+v", a, b)
	}
}
