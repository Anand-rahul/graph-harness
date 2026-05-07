// Package daemon owns workspace + daemon lifecycle: workspace discovery
// (walk up to a `.graph-harness` directory), pidfile + socket path derivation,
// and the watchman-style lazy-spawn entry point.
package daemon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Workspace describes the active graph-harness workspace.
type Workspace struct {
	Root        string // absolute path to repo root containing .graph-harness/
	StateDir    string // <Root>/.graph-harness
	OverlayDir  string // <Root>/.graph-harness/overlay
	PoliciesDir string // <Root>/.graph-harness/policies
	ConfigPath  string // <Root>/.graph-harness/graph-harness.toml
	EventLog    string // <RuntimeDir>/<workspace-id>.db
	SocketPath  string // <RuntimeDir>/<workspace-id>.sock  (linux/macos)
	PidFile     string // <RuntimeDir>/<workspace-id>.pid
	ID          string // hash of Root, used in socket / pid / db filenames
}

// Discover locates the workspace from cwd by walking up until a .graph-harness
// directory is found. If none exists, the returned Workspace.Root is "".
func Discover() (*Workspace, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		candidate := filepath.Join(dir, ".graph-harness")
		if _, err := os.Stat(candidate); err == nil {
			return From(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return &Workspace{}, nil
}

// From builds a Workspace value rooted at the given path. The directory does
// not need to contain `.graph-harness/` yet — `init` will create it.
func From(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	id := workspaceID(abs)
	rt, err := runtimeDir()
	if err != nil {
		return nil, err
	}
	w := &Workspace{
		Root:        abs,
		StateDir:    filepath.Join(abs, ".graph-harness"),
		OverlayDir:  filepath.Join(abs, ".graph-harness", "overlay"),
		PoliciesDir: filepath.Join(abs, ".graph-harness", "policies"),
		ConfigPath:  filepath.Join(abs, ".graph-harness", "graph-harness.toml"),
		EventLog:    filepath.Join(rt, id+".db"),
		PidFile:     filepath.Join(rt, id+".pid"),
		ID:          id,
	}
	w.SocketPath = socketPath(rt, id)
	return w, nil
}

// EnsureStateDir creates the on-disk workspace skeleton (.graph-harness/{overlay,policies}).
// Idempotent.
func (w *Workspace) EnsureStateDir() error {
	for _, d := range []string{w.StateDir, w.OverlayDir, w.PoliciesDir} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}

// IsInitialized reports whether `init` has already run for this workspace.
func (w *Workspace) IsInitialized() bool {
	if w.StateDir == "" {
		return false
	}
	_, err := os.Stat(w.StateDir)
	return err == nil
}

func workspaceID(root string) string {
	h := sha256.Sum256([]byte(root))
	return hex.EncodeToString(h[:6]) // 12-char hex; collisions astronomically rare per machine
}

func runtimeDir() (string, error) {
	switch runtime.GOOS {
	case "linux":
		if v := os.Getenv("XDG_RUNTIME_DIR"); v != "" {
			d := filepath.Join(v, "graph-harness")
			if err := os.MkdirAll(d, 0o700); err != nil { //nolint:gosec // runtime dir under user control
				return "", err
			}
			return d, nil
		}
		fallthrough
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		d := filepath.Join(home, "Library", "Application Support", "graph-harness")
		if runtime.GOOS == "linux" {
			d = filepath.Join(home, ".cache", "graph-harness")
		}
		if err := os.MkdirAll(d, 0o700); err != nil { //nolint:gosec // runtime dir under user control
			return "", err
		}
		return d, nil
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
		d := filepath.Join(base, "graph-harness")
		if err := os.MkdirAll(d, 0o700); err != nil { //nolint:gosec // runtime dir under user control
			return "", err
		}
		return d, nil
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func socketPath(rt, id string) string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\graph-harness-` + id
	}
	return filepath.Join(rt, id+".sock")
}
