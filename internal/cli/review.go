package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamstaq/graph-harness/internal/daemon"
	"github.com/shivamstaq/graph-harness/internal/review_queue"
)

// openReviewQueue colocates the review queue store with the event log under
// the runtime dir.
func openReviewQueue(ws *daemon.Workspace) (*review_queue.Queue, *sql.DB, error) {
	dsn := ws.EventLog + ".review.queue?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	q, err := review_queue.NewQueue(db)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return q, db, nil
}

// newReviewCmdReal replaces the stub. P0.T40.
func newReviewCmdReal() *cobra.Command {
	c := &cobra.Command{Use: "review", Short: "Inspect and resolve the review queue"}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pending review items",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			q, db, err := openReviewQueue(ws)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()
			items, err := q.List(context.Background(), "")
			if err != nil {
				return err
			}
			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
			}
			out := cmd.OutOrStdout()
			if len(items) == 0 {
				_, _ = fmt.Fprintln(out, "(no proposals in the review queue)")
				return nil
			}
			for _, p := range items {
				_, _ = fmt.Fprintf(out, "%-12s %-20s %-10s %s\n", p.ID, p.TargetLayer, p.State, p.Kind)
			}
			return nil
		},
	}
	listCmd.Flags().Bool("json", false, "emit JSON")

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Show one review item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			q, db, err := openReviewQueue(ws)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()
			p, err := q.Get(context.Background(), args[0])
			if err != nil {
				return err
			}
			if p == nil {
				return fmt.Errorf("proposal %s not found", args[0])
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(p)
		},
	}

	acceptCmd := &cobra.Command{
		Use:   "accept <id>",
		Short: "Accept a proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			q, db, err := openReviewQueue(ws)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()
			if err := q.Accept(context.Background(), args[0]); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ accepted %s\n", args[0])
			return nil
		},
	}

	rejectCmd := &cobra.Command{
		Use:   "reject <id>",
		Short: "Reject a proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			q, db, err := openReviewQueue(ws)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()
			if err := q.Reject(context.Background(), args[0]); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ rejected %s\n", args[0])
			return nil
		},
	}

	submitCmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit a proposal (test-only convenience; agents go through MCP)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			q, db, err := openReviewQueue(ws)
			if err != nil {
				return err
			}
			defer func() { _ = db.Close() }()
			layer, _ := cmd.Flags().GetString("layer")
			kind, _ := cmd.Flags().GetString("kind")
			author, _ := cmd.Flags().GetString("author")
			payload, _ := cmd.Flags().GetString("payload")
			id, err := q.Submit(context.Background(), review_queue.Proposal{
				TargetLayer: layer, Kind: kind, Author: author, Payload: []byte(payload),
			})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), id)
			return nil
		},
	}
	submitCmd.Flags().String("layer", "semantic.overlay", "target layer")
	submitCmd.Flags().String("kind", "selector", "proposal kind")
	submitCmd.Flags().String("author", "agent:test", "author tag")
	submitCmd.Flags().String("payload", "", "JSON payload")

	c.AddCommand(listCmd, getCmd, acceptCmd, rejectCmd, submitCmd)
	return c
}
