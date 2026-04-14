package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newArtifactCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "List and download run artifacts",
	}

	cmd.AddCommand(
		newArtifactListCmd(opts),
		newArtifactDownloadCmd(opts),
	)
	return cmd
}

func newArtifactListCmd(opts *rootOptions) *cobra.Command {
	var (
		taskID    string
		stepID    string
		pageSize  int
		pageToken string
	)

	cmd := &cobra.Command{
		Use:   "list <run-id>",
		Short: "List artifacts for one run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			query := url.Values{}
			if taskID != "" {
				query.Set("taskId", taskID)
			}
			if stepID != "" {
				query.Set("stepId", stepID)
			}
			if pageSize > 0 {
				query.Set("pageSize", fmt.Sprintf("%d", pageSize))
			}
			if pageToken != "" {
				query.Set("pageToken", pageToken)
			}

			var resp listRunArtifactsResponse
			if err := httpClient.GetJSONWithQuery(ctx, "/v1/batch/workflow-runs/"+args[0]+"/artifacts", query, &resp); err != nil {
				return err
			}

			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			return printArtifacts(cmd.OutOrStdout(), resp.Artifacts)
		},
	}
	cmd.Flags().StringVar(&taskID, "task-id", "", "Filter by task ID")
	cmd.Flags().StringVar(&stepID, "step-id", "", "Filter by step ID")
	cmd.Flags().IntVar(&pageSize, "page-size", 50, "Page size")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "Page token")
	return cmd
}

func newArtifactDownloadCmd(opts *rootOptions) *cobra.Command {
	var (
		outputDir string
		taskID    string
		stepID    string
	)

	cmd := &cobra.Command{
		Use:   "download <run-id>",
		Short: "Download all accessible artifacts for one run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}

			baseDir := outputDir
			if baseDir == "" {
				baseDir = "artifacts-" + args[0]
			}

			downloads := make([]map[string]any, 0)
			nextPageToken := ""
			for {
				ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
				query := url.Values{}
				query.Set("pageSize", "200")
				if taskID != "" {
					query.Set("taskId", taskID)
				}
				if stepID != "" {
					query.Set("stepId", stepID)
				}
				if nextPageToken != "" {
					query.Set("pageToken", nextPageToken)
				}

				var resp listRunArtifactsResponse
				err := httpClient.GetJSONWithQuery(ctx, "/v1/batch/workflow-runs/"+args[0]+"/artifacts", query, &resp)
				cancel()
				if err != nil {
					return err
				}

				for _, artifact := range resp.Artifacts {
					filename := inferArtifactFilename(artifact)
					targetPath, err := resolveFilePath(filepath.Join(baseDir, filename), filename)
					if err != nil {
						return err
					}

					var data []byte
					switch {
					case artifact.InlineText != "":
						data = []byte(artifact.InlineText)
					case artifact.AccessURL != "":
						downloadCtx, downloadCancel := context.WithTimeout(cmd.Context(), opts.timeout)
						data, err = downloadURL(downloadCtx, artifact.AccessURL)
						downloadCancel()
						if err != nil {
							return err
						}
					default:
						continue
					}

					if err := os.WriteFile(targetPath, data, 0o644); err != nil {
						return err
					}
					downloads = append(downloads, map[string]any{
						"artifactId": artifact.ArtifactID,
						"taskId":     artifact.TaskID,
						"stepId":     artifact.StepID,
						"mimeType":   artifact.MimeType,
						"path":       targetPath,
						"bytes":      len(data),
					})
				}

				nextPageToken = resp.NextPageToken
				if nextPageToken == "" {
					break
				}
			}

			result := map[string]any{
				"runId":     args[0],
				"downloads": downloads,
			}
			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			tw := newTabWriter(cmd.OutOrStdout())
			if _, err := fmt.Fprintln(tw, "artifact_id\ttask_id\tstep_id\tmime_type\tpath\tbytes"); err != nil {
				return err
			}
			for _, item := range downloads {
				if _, err := fmt.Fprintf(tw, "%v\t%v\t%v\t%v\t%v\t%v\n", item["artifactId"], item["taskId"], item["stepId"], item["mimeType"], item["path"], item["bytes"]); err != nil {
					return err
				}
			}
			return tw.Flush()
		},
	}
	cmd.Flags().StringVarP(&outputDir, "output-dir", "d", "", "Directory to save artifacts into")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Filter by task ID")
	cmd.Flags().StringVar(&stepID, "step-id", "", "Filter by step ID")
	return cmd
}
