package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamstaq/graph-harness/internal/bench"
	"github.com/shivamstaq/graph-harness/internal/change_process"
	"github.com/shivamstaq/graph-harness/internal/facts"
)

// newBenchCmdReal implements `graph-harness bench`. P0.T48 — scenario 1 only.
func newBenchCmdReal() *cobra.Command {
	c := &cobra.Command{
		Use:   "bench",
		Short: "Run the bench corpus against the oracle",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ws, err := activeWorkspace()
			if err != nil {
				return err
			}
			scenarioID, _ := cmd.Flags().GetInt("scenario")
			regime, _ := cmd.Flags().GetString("regime")
			if scenarioID == 0 {
				scenarioID = 1
			}
			if regime == "" {
				regime = "mature"
			}
			scenario := bench.Scenario{
				ID:          scenarioID,
				Name:        "auth-sensitive edit (Go)",
				Regime:      regime,
				Description: "Phase-0 fixture: edit a flow-scoped function without acknowledging the flow",
				Languages:   []string{"go"},
			}
			expected := []bench.ExpectedFinding{
				{Kind: "flow_unreviewed", Severity: "medium", Flow: "CheckoutValidation"},
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
			pipeline := &change_process.Pipeline{Overlay: overlay, Code: store}

			// Phase 0 scenario 1 uses a sentinel diff that touches the
			// flow-scoped function. The fixture lives at demo.diff in the
			// workspace; bench corpus authoring (P0.T47 polish) replaces
			// this with first-class scenario assets.
			diff := []byte(`--- a/internal/checkout/validator.go
+++ b/internal/checkout/validator.go
@@ -3,5 +3,7 @@ package checkout
 type CheckoutValidator struct{}

-func (v *CheckoutValidator) Validate(cart any) error {
+// Validate runs the cart validation pipeline (now with extra logging).
+func (v *CheckoutValidator) Validate(cart any) error {
+	_ = cart
 	return nil
 }
`)
			res, err := pipeline.ValidateDiff(ctx, diff, log.LastSeq())
			if err != nil {
				return err
			}
			score := bench.Score(scenario, res, expected)
			out := cmd.OutOrStdout()
			b, _ := score.AsJSON()
			_, _ = fmt.Fprintln(out, string(b))
			return nil
		},
	}
	c.Flags().Int("scenario", 0, "run a single scenario by ID")
	c.Flags().String("regime", "mature", "regime to run (mature only in P0)")
	c.Flags().String("gate", "", "evaluate against a named ship-bar (P6+)")
	return c
}
