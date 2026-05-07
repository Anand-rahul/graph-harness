package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/shivamstaq/graph-harness/internal/code_core"
	"github.com/shivamstaq/graph-harness/internal/daemon"
	"github.com/shivamstaq/graph-harness/internal/facts"
	"github.com/shivamstaq/graph-harness/internal/semantic_overlay"
	"github.com/shivamstaq/graph-harness/internal/source_live"
)

// activeWorkspace returns the discovered workspace (must be initialized) or
// an error explaining that `init` needs to run first.
func activeWorkspace() (*daemon.Workspace, error) {
	ws, err := daemon.Discover()
	if err != nil {
		return nil, err
	}
	if !ws.IsInitialized() {
		return nil, errors.New("no .graph-harness workspace found in or above cwd; run `graph-harness init` first")
	}
	return ws, nil
}

// openCodeStore opens (or creates) the code.core SQLite store for the
// workspace. The store is colocated with the event log under the runtime dir.
func openCodeStore(ws *daemon.Workspace) (*code_core.Store, *sql.DB, error) {
	dsn := ws.EventLog + ".code.core?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	store, err := code_core.NewStore(db)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return store, db, nil
}

// indexWorkspaceCode walks the workspace, parses every .go file, and ingests
// entities into code.core. Idempotent: re-running is cheap because identity
// is content-addressable.
func indexWorkspaceCode(ctx context.Context, ws *daemon.Workspace, store *code_core.Store, log *facts.EventLog) error {
	root := ws.Root
	files, err := goFilesUnder(root)
	if err != nil {
		return err
	}
	for _, abs := range files {
		// #nosec G304 -- abs originates from goFilesUnder under workspace root
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, abs)
		pf, err := source_live.ParseGoFile(rel, data)
		if err != nil {
			continue
		}
		seq := uint64(0)
		if log != nil {
			seq = log.LastSeq()
		}
		if _, err := store.IngestParsedFile(ctx, pf, seq); err != nil {
			return err
		}
	}
	return nil
}

func goFilesUnder(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // tolerate transient errors
		}
		if d.IsDir() {
			// Skip hidden + vendored + build dirs.
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "bin" || name == "dist" {
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

// loadOverlay reads .graph-harness/overlay/**/*.gh into an Overlay.
func loadOverlay(ws *daemon.Workspace) (*semantic_overlay.Overlay, error) {
	o := semantic_overlay.NewOverlay()
	errs, err := o.Load(ws.OverlayDir)
	if err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		// Surface the first parse error to the user but return the partial
		// overlay so they can see what *did* parse.
		for path, e := range errs {
			return o, fmt.Errorf("%s: %w", path, e)
		}
	}
	return o, nil
}
