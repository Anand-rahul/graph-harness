package kernel

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest is the parsed YAML manifest declared per-layer (SPEC §5.1).
// The validator rejects non-conforming primitives at install time.
type Manifest struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   ManifestMetadata `yaml:"metadata"`
	Schema     ManifestSchema   `yaml:"schema"`
	// Capabilities is a heterogeneous list (some entries are strings, some
	// are maps with their own subkeys). We accept `any` to keep the YAML
	// shape aligned with SPEC §5.3 and validate on a closed vocabulary in
	// ValidateCapabilities.
	Capabilities []any           `yaml:"capabilities"`
	Events       ManifestEvents  `yaml:"events"`
	Storage      ManifestStorage `yaml:"storage"`
	Snapshot     ManifestSnap    `yaml:"snapshot"`
	Trust        ManifestTrust   `yaml:"trust"`
	Sync         ManifestSync    `yaml:"sync"`

	// Optional layer-specific extensions (review.queue declares
	// promotion_mode, layers declare freshness_states, etc.).
	PromotionMode   string   `yaml:"promotion_mode,omitempty"`
	FreshnessStates []string `yaml:"freshness_states,omitempty"`
}

// ManifestMetadata describes the layer identity and dependency graph edges.
type ManifestMetadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	DependsOn   []string `yaml:"depends_on"`
	Description string   `yaml:"description,omitempty"`
}

// ManifestSchema declares entity and relation kinds. Empty slices are valid
// (working subset, not placeholder — e.g. code.framework in P0).
type ManifestSchema struct {
	EntityKinds   []EntityKindDecl   `yaml:"entity_kinds"`
	RelationKinds []RelationKindDecl `yaml:"relation_kinds"`
}

// EntityKindDecl declares one kind plus its attribute schema.
type EntityKindDecl struct {
	Name       string                  `yaml:"name"`
	Attributes map[string]AttributeDef `yaml:"attributes"`
}

// RelationKindDecl declares one relation kind.
type RelationKindDecl struct {
	Name       string                  `yaml:"name"`
	From       string                  `yaml:"from"`
	To         string                  `yaml:"to"`
	Attributes map[string]AttributeDef `yaml:"attributes"`
}

// AttributeDef constrains an attribute to the closed primitive set in
// SPEC §5.2. The validator rejects foreign types (`entity_ref(<other_layer>)`,
// arbitrary Go types, etc.).
type AttributeDef struct {
	Type    string `yaml:"type"`
	Indexed bool   `yaml:"indexed"`
}

// ManifestEvents lists the event kinds the layer is allowed to emit.
type ManifestEvents struct {
	Emits []string `yaml:"emits"`
}

// ManifestStorage describes the layer's adapter contract.
type ManifestStorage struct {
	Adapter    string `yaml:"adapter"`
	Durability string `yaml:"durability"`
	Mutability string `yaml:"mutability"`
}

// ManifestSnap declares snapshot semantics (SPEC §5.6).
type ManifestSnap struct {
	SupportsAsOfSeq bool          `yaml:"supports_as_of_seq"`
	Strategy        string        `yaml:"strategy"`
	Retention       SnapRetention `yaml:"retention"`
}

// SnapRetention is the per-layer event/duration retention floor.
type SnapRetention struct {
	Events   int    `yaml:"events"`
	Duration string `yaml:"duration"`
}

// ManifestTrust enumerates who may write directly versus via review.
type ManifestTrust struct {
	DirectWritesFrom   []string `yaml:"direct_writes_from"`
	ReviewRequiredFrom []string `yaml:"review_required_from"`
	ForbiddenFrom      []string `yaml:"forbidden_from"`
}

// ManifestSync mirrors SPEC §6.6.
type ManifestSync struct {
	Mode               string `yaml:"mode"`
	ConflictResolution string `yaml:"conflict_resolution,omitempty"`
	Peer               string `yaml:"peer,omitempty"`
}

// ParseManifest parses the YAML payload into a Manifest. It does not
// run schema validation; callers chain Validate() to enforce SPEC §5.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(false) // tolerate harmless extras; validator rejects schema violations
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// Validate enforces SPEC §5 closed vocabularies.
func (m *Manifest) Validate() error {
	if m.APIVersion != "graph-harness.kernel/v1" {
		return fmt.Errorf("apiVersion must be 'graph-harness.kernel/v1', got %q", m.APIVersion)
	}
	if m.Kind != "Layer" {
		return fmt.Errorf("kind must be 'Layer', got %q", m.Kind)
	}
	if m.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if m.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}
	for _, ek := range m.Schema.EntityKinds {
		for attrName, attr := range ek.Attributes {
			if err := validatePrimitive(attr.Type); err != nil {
				return fmt.Errorf("entity_kind %q attribute %q: %w", ek.Name, attrName, err)
			}
		}
	}
	for _, rk := range m.Schema.RelationKinds {
		for attrName, attr := range rk.Attributes {
			if err := validatePrimitive(attr.Type); err != nil {
				return fmt.Errorf("relation_kind %q attribute %q: %w", rk.Name, attrName, err)
			}
		}
	}
	return nil
}

// allowedPrimitives is the closed set from SPEC §5.2.
var allowedPrimitives = map[string]struct{}{
	"string":           {},
	"int":              {},
	"float":            {},
	"bool":             {},
	"timestamp":        {},
	"enum":             {},
	"path":             {},
	"selector":         {},
	"opaque_blob":      {},
	"entity_ref(self)": {},
}

// validatePrimitive enforces the closed attribute primitive set, including
// the keystone rule that `entity_ref(<other_layer>)` is forbidden — only
// `entity_ref(self)` is allowed; cross-layer references must use selectors.
func validatePrimitive(t string) error {
	t = strings.TrimSpace(t)
	if _, ok := allowedPrimitives[t]; ok {
		return nil
	}
	// list<T> and map<K,V> compose primitives; accept the form, validate inner.
	if strings.HasPrefix(t, "list<") && strings.HasSuffix(t, ">") {
		inner := strings.TrimSuffix(strings.TrimPrefix(t, "list<"), ">")
		return validatePrimitive(inner)
	}
	if strings.HasPrefix(t, "map<") && strings.HasSuffix(t, ">") {
		inner := strings.TrimSuffix(strings.TrimPrefix(t, "map<"), ">")
		parts := strings.SplitN(inner, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("malformed map type: %q", t)
		}
		if err := validatePrimitive(strings.TrimSpace(parts[0])); err != nil {
			return err
		}
		return validatePrimitive(strings.TrimSpace(parts[1]))
	}
	if strings.HasPrefix(t, "entity_ref(") && strings.HasSuffix(t, ")") {
		// Cross-layer entity_ref is forbidden; only entity_ref(self) allowed.
		return fmt.Errorf("entity_ref(<other_layer>) is forbidden — use a selector for cross-layer references (SPEC §5.2)")
	}
	return fmt.Errorf("unknown attribute primitive %q (allowed: %s)", t, allowedPrimitiveList())
}

func allowedPrimitiveList() string {
	s := make([]string, 0, len(allowedPrimitives))
	for p := range allowedPrimitives {
		s = append(s, p)
	}
	return strings.Join(s, ", ") + ", list<T>, map<K,V>"
}
