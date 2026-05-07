package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamstaq/graph-harness/internal/facts"
)

// newTUICmdReal implements `graph-harness tui`. Minimal P0.T41 skeleton:
// exits cleanly (no TTY check yet) and prints a workspace status snapshot.
// Full bubbletea cockpit is fleshed out in subsequent commits.
func newTUICmdReal() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the terminal cockpit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			log, err := facts.OpenEventLog(ws.EventLog)
			if err != nil {
				return err
			}
			defer func() { _ = log.Close() }()

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintln(out, "─── graph-harness TUI (read-only skeleton) ──────")
			_, _ = fmt.Fprintf(out, "Workspace: %s\n", ws.Root)
			_, _ = fmt.Fprintf(out, "Event log: %s (last seq=%d)\n", ws.EventLog, log.LastSeq())
			_, _ = fmt.Fprintln(out, "Findings:  (live cockpit lands with bubbletea event subscription)")
			_, _ = fmt.Fprintln(out, "Press Ctrl-C to exit")

			// Honor the "interactive" flag: when not set, exit cleanly so
			// gotit specs can drive the surface without a PTY.
			interactive, _ := cmd.Flags().GetBool("interactive")
			if !interactive {
				return nil
			}
			// Block on context for live mode; full bubbletea implementation lands
			// alongside the event subscription wiring.
			<-context.Background().Done()
			return nil
		},
	}
}
