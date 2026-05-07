package kernel

import (
	"slices"
	"sort"
)

// FoldProvenance applies the kernel fold rule from SPEC §4.5:
//
//	confidence    = min(confidence_i)
//	freshness     = worst(freshness_i)        // by foldOrder
//	source_class  = union(source_class_i)
//	inputs        = deduped_union(inputs_i)
//	produced_seq  = max(produced_seq_i)
//
// Constituents are preserved on the result so callers can inspect detail.
// Empty input returns the zero value (caller decides if that's an error).
func FoldProvenance(in []Provenance) Provenance {
	if len(in) == 0 {
		return Provenance{}
	}

	out := Provenance{
		Confidence:   1.0,
		Freshness:    in[0].Freshness,
		ProducedSeq:  0,
		Constituents: append([]Provenance(nil), in...),
	}

	srcSet := make(map[SourceClass]struct{})
	inSet := make(map[string]struct{})
	freshnessRisk := -1

	for i, p := range in {
		if i == 0 || p.Confidence < out.Confidence {
			out.Confidence = p.Confidence
		}
		if r := BlockingRisk(p.Freshness); r > freshnessRisk {
			freshnessRisk = r
			out.Freshness = p.Freshness
		}
		if p.ProducedSeq > out.ProducedSeq {
			out.ProducedSeq = p.ProducedSeq
		}
		for _, s := range p.SourceClass {
			srcSet[s] = struct{}{}
		}
		for _, in := range p.Inputs {
			inSet[in] = struct{}{}
		}
	}

	out.SourceClass = make([]SourceClass, 0, len(srcSet))
	for s := range srcSet {
		out.SourceClass = append(out.SourceClass, s)
	}
	slices.Sort(out.SourceClass)

	out.Inputs = make([]string, 0, len(inSet))
	for in := range inSet {
		out.Inputs = append(out.Inputs, in)
	}
	sort.Strings(out.Inputs)

	return out
}
