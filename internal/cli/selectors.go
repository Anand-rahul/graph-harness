package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamstaq/graph-harness/internal/change_process"
	"github.com/shivamstaq/graph-harness/internal/facts"
)

// newSelectorsTestCmd implements `graph-harness selectors test <name>` (P0.T27).
func newSelectorsTestCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "test <name>",
		Short: "Resolve a named selector and print its outcome envelope",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			ctx := context.Background()

			log, err := facts.OpenEventLog(ws.EventLog)
			if err != nil {
				return err
			}
			defer func() { _ = log.Close() }()

			store, db, err := openCodeStore(ws)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()

			if err := indexWorkspaceCode(ctx, ws, store, log); err != nil {
				return err
			}
			overlay, err := loadOverlay(ws)
			if err != nil {
				return err
			}
			env, err := overlay.Resolve(ctx, args[0], store, log.LastSeq())
			if err != nil {
				return err
			}
			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(env)
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "selector %s\n", env.SelectorID)
			_, _ = fmt.Fprintf(out, "  outcome:    %s\n", env.Outcome)
			_, _ = fmt.Fprintf(out, "  matches:    %d\n", len(env.Matches))
			for _, m := range env.Matches {
				_, _ = fmt.Fprintf(out, "    - id:               %s\n", m.EntityID)
				_, _ = fmt.Fprintf(out, "      qualified_name:   %s\n", m.QualifiedName)
				_, _ = fmt.Fprintf(out, "      confidence:       %.2f\n", m.Confidence)
				_, _ = fmt.Fprintf(out, "      via_anchor:       %s\n", m.ViaAnchor)
			}
			_, _ = fmt.Fprintf(out, "  resolved_at: kernel_event_seq=%d\n", env.ResolvedAt)
			return nil
		},
	}
	c.Flags().Bool("json", false, "emit envelope as JSON")
	return c
}

// newFlowsListCmd implements `graph-harness flows list` (P0.T27).
func newFlowsListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List flows declared in the semantic overlay",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			overlay, err := loadOverlay(ws)
			if err != nil {
				return err
			}
			asJSON, _ := cmd.Flags().GetBool("json")
			names := make([]string, 0, len(overlay.Flows))
			for n := range overlay.Flows {
				names = append(names, n)
			}
			sort.Strings(names)
			if asJSON {
				rows := make([]map[string]any, 0, len(names))
				for _, n := range names {
					f := overlay.Flows[n]
					rows = append(rows, map[string]any{
						"name":        f.Name,
						"description": f.Description,
						"scope":       f.Scope,
						"steps":       len(f.Steps),
					})
				}
				return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
			}
			for _, n := range names {
				f := overlay.Flows[n]
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-30s %d steps\n", f.Name, len(f.Steps))
			}
			return nil
		},
	}
	c.Flags().Bool("json", false, "emit JSON")
	return c
}

// newValidateDiffCmdReal builds the real RunE for `graph-harness validate-diff`.
// Wrapped in stubs.go to compose flag definitions.
func newValidateDiffCmdReal() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-diff",
		Short: "Run the change.process pipeline against a unified diff",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			ctx := context.Background()
			diffPath, _ := cmd.Flags().GetString("diff")
			var unified []byte
			switch diffPath {
			case "", "-":
				unified, err = io.ReadAll(cmd.InOrStdin())
			default:
				// #nosec G304 -- operator-supplied path; standard CLI behavior.
				unified, err = os.ReadFile(diffPath)
			}
			if err != nil {
				return err
			}
			if strings.TrimSpace(string(unified)) == "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "0 findings (empty diff)")
				return nil
			}
			log, err := facts.OpenEventLog(ws.EventLog)
			if err != nil {
				return err
			}
			defer func() { _ = log.Close() }()
			store, db, err := openCodeStore(ws)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()
			if err := indexWorkspaceCode(ctx, ws, store, log); err != nil {
				return err
			}
			overlay, err := loadOverlay(ws)
			if err != nil {
				return err
			}
			pipeline := &change_process.Pipeline{Overlay: overlay, Code: store}
			res, err := pipeline.ValidateDiff(ctx, unified, log.LastSeq())
			if err != nil {
				return err
			}
			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintln(out, "─── Validation Result ──────────────────────────")
			_, _ = fmt.Fprintf(out, "%s\n\n", res.Summary)
			for _, f := range res.Findings {
				_, _ = fmt.Fprintf(out, "▼ %s — %s\n", f.ID, f.Kind)
				_, _ = fmt.Fprintf(out, "  Subject: %s (%s)\n", f.Subject.Qualified, f.Subject.EntityKind)
				_, _ = fmt.Fprintf(out, "  Flow:    %s\n", f.Subject.Flow)
				_, _ = fmt.Fprintln(out, "  Evidence:")
				for _, e := range f.Evidence {
					_, _ = fmt.Fprintf(out, "    - %s: %s\n", e.Kind, e.Detail)
				}
				_, _ = fmt.Fprintln(out, "  Repair:  (deferred to P3)")
			}
			_, _ = fmt.Fprintln(out, "─────────────────────────────────────────────────")
			return nil
		},
	}
}
