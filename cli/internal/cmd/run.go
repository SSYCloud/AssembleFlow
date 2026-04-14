package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newRunCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Submit and monitor template runs",
	}

	cmd.AddCommand(
		newRunSubmitCmd(opts),
		newRunWatchCmd(opts),
	)
	return cmd
}

func newRunSubmitCmd(opts *rootOptions) *cobra.Command {
	var (
		inputPath      string
		callbackURL    string
		idempotencyKey string
	)

	cmd := &cobra.Command{
		Use:   "submit <template-id>",
		Short: "Validate, precheck, and submit official template rows from JSON or JSONL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if inputPath == "" {
				return fmt.Errorf("--file is required")
			}

			rows, err := loadTemplateRows(inputPath)
			if err != nil {
				return err
			}

			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			var schemaResp templateSchemaResponse
			if err := httpClient.GetJSON(ctx, "/v1/templates/"+args[0]+"/schema", &schemaResp); err != nil {
				return err
			}
			rows = remapRowsToHeaderLabels(rows, schemaResp)

			payload := map[string]any{
				"templateId": args[0],
				"rows":       rows,
			}
			if callbackURL != "" {
				payload["callbackUrl"] = callbackURL
			}
			if idempotencyKey != "" {
				payload["idempotencyKey"] = idempotencyKey
			}

			var validateResp validateTemplateRowsResponse
			if err := httpClient.PostJSON(ctx, "/v1/templates:validate-rows", payload, &validateResp); err != nil {
				return err
			}
			if !validateResp.Valid {
				if opts.output == "json" {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					_ = enc.Encode(map[string]any{
						"templateId": args[0],
						"inputPath":  inputPath,
						"validation": validateResp,
					})
				}
				return validationError(validateResp)
			}

			var precheckResp precheckTemplateRowsResponse
			if err := httpClient.PostJSON(ctx, "/v1/templates:precheck-rows", payload, &precheckResp); err != nil {
				return err
			}
			if balance := precheckResp.BalanceCheck; balance != nil && !balance.IsSufficient {
				return fmt.Errorf("insufficient balance: estimated_cost=%s available=%s", formatCost(int64(precheckResp.EstimatedTotalCost)), formatCost(int64(balance.AvailableBalance)))
			}

			var submitResp submitTemplateRowsResponse
			if err := httpClient.PostJSON(ctx, "/v1/templates:submit-rows", payload, &submitResp); err != nil {
				return err
			}

			result := map[string]any{
				"templateId":         args[0],
				"inputPath":          inputPath,
				"rowCount":           len(rows),
				"estimatedTotalCost": int64(precheckResp.EstimatedTotalCost),
				"balanceCheck":       precheckResp.BalanceCheck,
				"runId":              submitResp.RunID,
				"status":             submitResp.Status,
				"acceptedAt":         int64(submitResp.AcceptedAt),
			}

			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			if err := printPrecheck(cmd.OutOrStdout(), precheckResp); err != nil {
				return err
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"template_id\t%s\ninput_path\t%s\nrow_count\t%d\nrun_id\t%s\nstatus\t%s\naccepted_at\t%s\n",
				args[0],
				inputPath,
				len(rows),
				submitResp.RunID,
				submitResp.Status,
				formatUnix(int64(submitResp.AcceptedAt)),
			)
			return err
		},
	}
	cmd.Flags().StringVarP(&inputPath, "file", "f", "", "Input file in JSON array or JSONL format")
	cmd.Flags().StringVar(&callbackURL, "callback-url", "", "Optional callback URL")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func newRunWatchCmd(opts *rootOptions) *cobra.Command {
	interval := 5 * time.Second

	cmd := &cobra.Command{
		Use:   "watch <run-id>",
		Short: "Poll a run until it reaches a terminal state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}

			var latest runStatusResponse
			for {
				ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
				err := httpClient.GetJSON(ctx, "/v1/batch/workflow-runs/"+args[0], &latest)
				cancel()
				if err != nil {
					return err
				}

				if opts.output != "json" {
					_, err := fmt.Fprintf(
						cmd.OutOrStdout(),
						"status=%s completed=%d/%d failed=%d cancelled=%d cost=%s\n",
						latest.Status,
						int(latest.CompletedTasks),
						int(latest.TotalTasks),
						int(latest.FailedTasks),
						int(latest.CancelledTasks),
						formatCost(int64(latest.ActualCost)),
					)
					if err != nil {
						return err
					}
				}

				if isTerminalRunStatus(latest.Status) {
					break
				}

				select {
				case <-cmd.Context().Done():
					return cmd.Context().Err()
				case <-time.After(interval):
				}
			}

			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(latest)
			}
			return printRunSummary(cmd.OutOrStdout(), latest)
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", interval, "Polling interval")
	return cmd
}
