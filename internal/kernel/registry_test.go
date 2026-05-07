package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifest_RejectsForeignRawIDEntityRef(t *testing.T) {
	yaml := `
apiVersion: graph-harness.kernel/v1
kind: Layer
metadata:
  name: bad.layer
  version: 0.1.0
schema:
  entity_kinds:
    - name: BadKind
      attributes:
        forbidden: { type: "entity_ref(other.layer)" }
  relation_kinds: []
storage:    { adapter: sqlite, durability: persistent, mutability: derived }
snapshot:   { supports_as_of_seq: false, strategy: event_replay, retention: { events: 0, duration: "0h" } }
trust:      { direct_writes_from: [], review_required_from: [], forbidden_from: [] }
events:     { emits: [] }
sync:       { mode: local_only }
`
	m, err := ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "entity_ref") {
		t.Fatalf("expected entity_ref rejection, got %v", err)
	}
}

func TestManifest_RejectsInvalidPrimitive(t *testing.T) {
	yaml := `
apiVersion: graph-harness.kernel/v1
kind: Layer
metadata: { name: bad, version: 0.1.0 }
schema:
  entity_kinds:
    - name: K
      attributes:
        a: { type: "wibble" }
  relation_kinds: []
storage:    { adapter: sqlite, durability: persistent, mutability: derived }
snapshot:   { supports_as_of_seq: false, strategy: event_replay, retention: { events: 0, duration: "0h" } }
trust:      { direct_writes_from: [], review_required_from: [], forbidden_from: [] }
events:     { emits: [] }
sync:       { mode: local_only }
`
	m, _ := ParseManifest([]byte(yaml))
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "unknown attribute primitive") {
		t.Fatalf("expected unknown primitive rejection, got %v", err)
	}
}

func TestManifest_RejectsMissingRequired(t *testing.T) {
	yaml := `
apiVersion: graph-harness.kernel/v1
kind: Layer
metadata: { version: 0.1.0 }
schema: { entity_kinds: [], relation_kinds: [] }
storage:    { adapter: sqlite, durability: persistent, mutability: derived }
snapshot:   { supports_as_of_seq: false, strategy: event_replay, retention: { events: 0, duration: "0h" } }
trust:      { direct_writes_from: [], review_required_from: [], forbidden_from: [] }
events:     { emits: [] }
sync:       { mode: local_only }
`
	m, _ := ParseManifest([]byte(yaml))
	if err := m.Validate(); err == nil || !strings.Contains(err.Error(), "metadata.name") {
		t.Fatalf("expected missing-name rejection, got %v", err)
	}
}

func TestRegistry_LoadAllProjectManifests(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("go.mod not found")
		}
		root = parent
	}
	reg := NewRegistry()
	if err := reg.LoadDir(filepath.Join(root, "manifests")); err != nil {
		t.Fatalf("load manifests: %v", err)
	}
	want := []string{
		"bench.oracle",
		"bench.scenarios",
		"change.process",
		"code.core",
		"code.framework",
		"history.evolution",
		"review.queue",
		"semantic.overlay",
		"source.live",
	}
	for _, name := range want {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("expected layer %q to be installed", name)
		}
	}
	// Topo order: source.live before code.core; code.core before semantic.overlay.
	order := reg.List()
	idx := func(s string) int {
		for i, n := range order {
			if n == s {
				return i
			}
		}
		return -1
	}
	if idx("source.live") >= idx("code.core") {
		t.Errorf("source.live must precede code.core: %v", order)
	}
	if idx("code.core") >= idx("semantic.overlay") {
		t.Errorf("code.core must precede semantic.overlay: %v", order)
	}
}
