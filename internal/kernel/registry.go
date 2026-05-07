package kernel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Registry tracks installed layers and enforces dependency-DAG topological
// ordering at install time. SPEC §2 + §5.
type Registry struct {
	layers map[string]*Manifest
}

// NewRegistry returns an empty layer registry.
func NewRegistry() *Registry {
	return &Registry{layers: map[string]*Manifest{}}
}

// Install validates the manifest and registers the layer. Cycles in the
// dependency graph are rejected.
func (r *Registry) Install(m *Manifest) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if _, exists := r.layers[m.Metadata.Name]; exists {
		return fmt.Errorf("layer %q already installed", m.Metadata.Name)
	}
	// Resolve depends_on: we accept names with optional `@<constraint>` suffix.
	for _, dep := range m.Metadata.DependsOn {
		depName := strings.SplitN(dep, "@", 2)[0]
		if _, ok := r.layers[depName]; !ok {
			return fmt.Errorf("layer %q depends on %q which is not installed", m.Metadata.Name, depName)
		}
	}
	r.layers[m.Metadata.Name] = m
	if err := r.checkAcyclic(); err != nil {
		// Roll back if (somehow) install introduced a cycle — depends_on
		// validation above usually prevents this, but defensive.
		delete(r.layers, m.Metadata.Name)
		return err
	}
	return nil
}

// Get returns a registered manifest by name.
func (r *Registry) Get(name string) (*Manifest, bool) {
	m, ok := r.layers[name]
	return m, ok
}

// List returns installed layer names in topological install order.
func (r *Registry) List() []string {
	out := r.topoSort()
	return out
}

// LoadDir installs every *.yaml manifest in the given directory in
// topologically valid order. Manifests with unmet deps wait until their
// deps land.
func (r *Registry) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read manifests dir %s: %w", dir, err)
	}
	pending := make(map[string]*Manifest)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		// #nosec G304 -- dir is operator-controlled.
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		m, err := ParseManifest(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		pending[m.Metadata.Name] = m
	}
	// Repeatedly install any manifest whose deps are satisfied.
	for len(pending) > 0 {
		progress := false
		// Iterate names sorted for determinism.
		names := make([]string, 0, len(pending))
		for n := range pending {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			m := pending[n]
			ready := true
			for _, dep := range m.Metadata.DependsOn {
				depName := strings.SplitN(dep, "@", 2)[0]
				if _, ok := r.layers[depName]; !ok {
					ready = false
					break
				}
			}
			if ready {
				if err := r.Install(m); err != nil {
					return fmt.Errorf("install %s: %w", n, err)
				}
				delete(pending, n)
				progress = true
			}
		}
		if !progress {
			missing := make([]string, 0, len(pending))
			for n := range pending {
				missing = append(missing, n)
			}
			sort.Strings(missing)
			return fmt.Errorf("dependency cycle or missing dep among: %s", strings.Join(missing, ", "))
		}
	}
	return nil
}

// checkAcyclic performs a Kahn's-algorithm-style topological scan; cycles
// would leave pending nodes when no zero-in-degree nodes remain.
func (r *Registry) checkAcyclic() error {
	indeg := make(map[string]int, len(r.layers))
	graph := make(map[string][]string, len(r.layers))
	for name, m := range r.layers {
		indeg[name] = 0
		_ = m // satisfy unused
	}
	for name, m := range r.layers {
		for _, dep := range m.Metadata.DependsOn {
			depName := strings.SplitN(dep, "@", 2)[0]
			if _, ok := r.layers[depName]; ok {
				graph[depName] = append(graph[depName], name)
				indeg[name]++
			}
		}
	}
	queue := make([]string, 0)
	for n, d := range indeg {
		if d == 0 {
			queue = append(queue, n)
		}
	}
	visited := 0
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		visited++
		for _, m := range graph[n] {
			indeg[m]--
			if indeg[m] == 0 {
				queue = append(queue, m)
			}
		}
	}
	if visited != len(r.layers) {
		return fmt.Errorf("dependency graph has a cycle")
	}
	return nil
}

// topoSort returns layer names in valid install order (deps before dependents).
func (r *Registry) topoSort() []string {
	indeg := make(map[string]int, len(r.layers))
	graph := make(map[string][]string, len(r.layers))
	names := make([]string, 0, len(r.layers))
	for name := range r.layers {
		indeg[name] = 0
		names = append(names, name)
	}
	for name, m := range r.layers {
		for _, dep := range m.Metadata.DependsOn {
			depName := strings.SplitN(dep, "@", 2)[0]
			if _, ok := r.layers[depName]; ok {
				graph[depName] = append(graph[depName], name)
				indeg[name]++
			}
		}
	}
	// Kahn with deterministic tiebreak (lexical).
	queue := make([]string, 0)
	for _, n := range names {
		if indeg[n] == 0 {
			queue = append(queue, n)
		}
	}
	sort.Strings(queue)
	out := make([]string, 0, len(r.layers))
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		out = append(out, n)
		next := graph[n]
		sort.Strings(next)
		for _, m := range next {
			indeg[m]--
			if indeg[m] == 0 {
				queue = append(queue, m)
			}
		}
		sort.Strings(queue)
	}
	return out
}
