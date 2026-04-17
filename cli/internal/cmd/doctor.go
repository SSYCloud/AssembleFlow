package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SSYCloud/loomloom/cli/internal/version"
	"github.com/spf13/cobra"
)

type healthResponse struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message"`
}

func newDoctorCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check LoomLoom server reachability and token wiring",
		RunE: func(cmd *cobra.Command, args []string) error {
			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			var resp healthResponse
			if err := httpClient.GetJSON(ctx, "/v1/health", &resp); err != nil {
				return err
			}

			versionStatus, versionErr := version.CheckLatest(ctx)
			currentVersion := version.Version
			latestVersion := ""
			updateAvailable := false
			upgradeHint := ""
			versionCheckError := ""
			if versionStatus != nil {
				currentVersion = versionStatus.CurrentVersion
				latestVersion = versionStatus.LatestVersion
				updateAvailable = versionStatus.UpdateAvailable
				upgradeHint = versionStatus.UpgradeHint
			}
			if versionErr != nil {
				versionCheckError = versionErr.Error()
			}

			if opts.output == "json" {
				payload := map[string]any{
					"server":              opts.server,
					"token_set":           opts.token != "",
					"healthy":             resp.Healthy,
					"message":             resp.Message,
					"cli_version":         currentVersion,
					"latest_release":      latestVersion,
					"update_available":    updateAvailable,
					"upgrade_hint":        upgradeHint,
					"version_check_error": versionCheckError,
					"base_usage":          "set LOOMLOOM_SERVER and LOOMLOOM_TOKEN before running template commands",
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(payload)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "server: %s\n", opts.server)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "token: %t\n", opts.token != "")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "healthy: %t\n", resp.Healthy)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "message: %s\n", resp.Message)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "cli_version: %s\n", currentVersion)
			if latestVersion != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "latest_release: %s\n", latestVersion)
			}
			if upgradeHint != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "notice: %s\n", upgradeHint)
			} else if versionCheckError != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "version_check: %s\n", versionCheckError)
			}
			return nil
		},
	}
}
