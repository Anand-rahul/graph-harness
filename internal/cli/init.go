package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shivamstaq/graph-harness/internal/daemon"
)

// newInitCmd implements `graph-harness init`. P0.T49 / demo §4.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a .graph-harness workspace in the current directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			ws, err := daemon.From(cwd)
			if err != nil {
				return err
			}
			if ws.IsInitialized() {
				return errors.New(".graph-harness/ already exists; this workspace is already initialized")
			}
			if err := ws.EnsureStateDir(); err != nil {
				return err
			}
			// Seed an empty workspace config (P0.T49 will populate from the
			// templates/workspace.toml.tmpl in a future iteration).
			if err := os.WriteFile(ws.ConfigPath, []byte("# graph-harness workspace config\n"), 0o600); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "✓ Created %s\n", filepath.Join(ws.StateDir, "{overlay,policies}"))
			_, _ = fmt.Fprintf(out, "✓ Workspace ID: %s\n", ws.ID)
			_, _ = fmt.Fprintf(out, "✓ Daemon socket: %s\n", ws.SocketPath)
			_, _ = fmt.Fprintf(out, "✓ Event log: %s\n", ws.EventLog)
			return nil
		},
	}
}
