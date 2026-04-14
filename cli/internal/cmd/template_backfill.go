package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xuri/excelize/v2"
)

const (
	templateMetaSheetName = "__batchjob_meta"
	templateDataSheetName = "data"
)

var templateResultHeaders = []string{
	"结果.运行状态",
	"结果.错误信息",
	"结果.text输出",
	"结果.image链接",
	"结果.video链接",
	"结果.task_id",
}

type templateBackfillRow struct {
	taskID      string
	status      string
	errorText   string
	taskIDs     map[string]struct{}
	textOutputs []string
	imageLink   string
	videoLink   string
}

func newTemplateBackfillResultsCmd(opts *rootOptions) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "backfill-results <run-id> <xlsx-path>",
		Short: "Backfill one official template workbook with run results",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID := strings.TrimSpace(args[0])
			workbookPath := strings.TrimSpace(args[1])

			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			var runResp runStatusResponse
			if err := httpClient.GetJSON(ctx, "/v1/batch/workflow-runs/"+runID, &runResp); err != nil {
				return err
			}

			tasks, err := listAllTasks(ctx, httpClient, runID)
			if err != nil {
				return err
			}

			artifacts, err := listAllArtifacts(ctx, httpClient, runID)
			if err != nil {
				return err
			}

			targetPath, err := resolveBackfillOutputPath(outputPath, workbookPath)
			if err != nil {
				return err
			}

			rowCount, err := backfillWorkbookResults(workbookPath, targetPath, tasks, artifacts)
			if err != nil {
				return err
			}

			result := map[string]any{
				"runId":         runID,
				"inputFile":     workbookPath,
				"outputFile":    targetPath,
				"status":        runResp.Status,
				"artifactCount": len(artifacts),
				"rowCount":      rowCount,
			}
			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"run_id\t%s\nstatus\t%s\ninput_file\t%s\noutput_file\t%s\nartifact_count\t%d\nrow_count\t%d\n",
				runID,
				runResp.Status,
				workbookPath,
				targetPath,
				len(artifacts),
				rowCount,
			)
			return err
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output-file", "f", "", "Optional output .xlsx path; defaults to overwriting the input workbook")
	return cmd
}

func listAllTasks(ctx context.Context, httpClient interface {
	GetJSONWithQuery(context.Context, string, url.Values, any) error
}, runID string) ([]runTaskEntry, error) {
	tasks := make([]runTaskEntry, 0)
	nextPageToken := ""
	for {
		query := url.Values{}
		query.Set("pageSize", "200")
		if nextPageToken != "" {
			query.Set("pageToken", nextPageToken)
		}

		var resp listRunTasksResponse
		if err := httpClient.GetJSONWithQuery(ctx, "/v1/batch/workflow-runs/"+runID+"/tasks", query, &resp); err != nil {
			return nil, err
		}
		tasks = append(tasks, resp.Tasks...)
		if resp.NextPageToken == "" {
			break
		}
		nextPageToken = resp.NextPageToken
	}
	return tasks, nil
}

func listAllArtifacts(ctx context.Context, httpClient interface {
	GetJSONWithQuery(context.Context, string, url.Values, any) error
}, runID string) ([]artifactEntry, error) {
	artifacts := make([]artifactEntry, 0)
	nextPageToken := ""
	for {
		query := url.Values{}
		query.Set("pageSize", "200")
		if nextPageToken != "" {
			query.Set("pageToken", nextPageToken)
		}

		var resp listRunArtifactsResponse
		if err := httpClient.GetJSONWithQuery(ctx, "/v1/batch/workflow-runs/"+runID+"/artifacts", query, &resp); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, resp.Artifacts...)
		if resp.NextPageToken == "" {
			break
		}
		nextPageToken = resp.NextPageToken
	}
	return artifacts, nil
}

func resolveBackfillOutputPath(outputPath string, workbookPath string) (string, error) {
	if strings.TrimSpace(outputPath) == "" {
		return filepath.Abs(workbookPath)
	}
	defaultName := strings.TrimSuffix(filepath.Base(workbookPath), filepath.Ext(workbookPath)) + ".result.xlsx"
	return resolveFilePath(outputPath, defaultName)
}

func backfillWorkbookResults(inputPath, outputPath string, tasks []runTaskEntry, artifacts []artifactEntry) (int, error) {
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return 0, fmt.Errorf("open workbook: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := validateWorkbookMetaSheet(f); err != nil {
		return 0, err
	}

	rows, err := f.GetRows(templateDataSheetName)
	if err != nil {
		return 0, fmt.Errorf("get data rows: %w", err)
	}
	if len(rows) < 1 {
		return 0, fmt.Errorf("data sheet has too few rows: got %d, expected at least 1 header row", len(rows))
	}
	headers := rows[0]
	if len(headers) == 0 {
		return 0, fmt.Errorf("data sheet row 1 (header) is empty")
	}

	resultCols := ensureResultColumns(f, templateDataSheetName, headers)
	rowCount := len(rows) - 1
	grouped, err := groupTasksAndArtifactsByRow(tasks, artifacts, rowCount)
	if err != nil {
		return 0, err
	}

	for rowIdx := 0; rowIdx < rowCount; rowIdx++ {
		entry := grouped[rowIdx]
		if entry == nil {
			entry = &templateBackfillRow{taskIDs: make(map[string]struct{})}
		}
		values := []any{
			localizeBackfillTaskStatus(entry.status),
			entry.errorText,
			strings.Join(entry.textOutputs, "\n---\n"),
			entry.imageLink,
			entry.videoLink,
			entry.primaryTaskID(),
		}
		for i, value := range values {
			cell, err := excelize.CoordinatesToCellName(resultCols[i], rowIdx+2)
			if err != nil {
				return 0, fmt.Errorf("result cell name row %d col %d: %w", rowIdx, i, err)
			}
			if err := f.SetCellValue(templateDataSheetName, cell, value); err != nil {
				return 0, fmt.Errorf("set result cell %s: %w", cell, err)
			}
		}
	}

	if err := f.SaveAs(outputPath); err != nil {
		return 0, fmt.Errorf("write backfilled workbook: %w", err)
	}
	return rowCount, nil
}

func validateWorkbookMetaSheet(f *excelize.File) error {
	metaRows, err := f.GetRows(templateMetaSheetName)
	if err != nil {
		return fmt.Errorf("get metadata rows: %w", err)
	}
	if len(metaRows) < 2 {
		return fmt.Errorf("metadata sheet %q has too few rows: got %d, expected at least 2", templateMetaSheetName, len(metaRows))
	}
	if len(metaRows[0]) < 2 || strings.TrimSpace(metaRows[0][0]) != "__template_id" || strings.TrimSpace(metaRows[0][1]) == "" {
		return fmt.Errorf("metadata sheet %q row 1 is invalid", templateMetaSheetName)
	}
	if len(metaRows[1]) < 2 || strings.TrimSpace(metaRows[1][0]) != "__template_version" || strings.TrimSpace(metaRows[1][1]) == "" {
		return fmt.Errorf("metadata sheet %q row 2 is invalid", templateMetaSheetName)
	}
	return nil
}

func ensureResultColumns(f *excelize.File, sheet string, headers []string) []int {
	headerToCol := make(map[string]int, len(headers))
	for idx, header := range headers {
		headerToCol[header] = idx + 1
	}
	nextCol := len(headers) + 1
	resultCols := make([]int, 0, len(templateResultHeaders))
	for _, header := range templateResultHeaders {
		col, ok := headerToCol[header]
		if !ok {
			col = nextCol
			nextCol++
			cell, _ := excelize.CoordinatesToCellName(col, 1)
			_ = f.SetCellValue(sheet, cell, header)
		}
		resultCols = append(resultCols, col)
	}
	return resultCols
}

func groupTasksAndArtifactsByRow(tasks []runTaskEntry, artifacts []artifactEntry, rowCount int) (map[int]*templateBackfillRow, error) {
	grouped := make(map[int]*templateBackfillRow)
	for _, task := range tasks {
		rowIdx := int(task.SourceRowIndex)
		if rowIdx < 0 {
			continue
		}
		if rowIdx >= rowCount {
			return nil, fmt.Errorf("task source_row_index %d exceeds workbook data rows %d", rowIdx, rowCount)
		}
		entry := grouped[rowIdx]
		if entry == nil {
			entry = &templateBackfillRow{taskIDs: make(map[string]struct{})}
			grouped[rowIdx] = entry
		}
		entry.taskID = task.TaskID
		entry.status = strings.TrimSpace(task.Status)
		entry.errorText = strings.TrimSpace(task.ErrorMessage)
		if task.TaskID != "" {
			entry.taskIDs[task.TaskID] = struct{}{}
		}
	}

	for _, artifact := range artifacts {
		rowIdx := int(artifact.SourceRowIndex)
		if rowIdx < 0 {
			continue
		}
		if rowIdx >= rowCount {
			return nil, fmt.Errorf("artifact source_row_index %d exceeds workbook data rows %d", rowIdx, rowCount)
		}
		entry := grouped[rowIdx]
		if entry == nil {
			entry = &templateBackfillRow{taskIDs: make(map[string]struct{})}
			grouped[rowIdx] = entry
		}
		if artifact.TaskID != "" {
			entry.taskIDs[artifact.TaskID] = struct{}{}
		}
		switch {
		case strings.HasPrefix(artifact.MimeType, "text/"):
			if text := strings.TrimSpace(artifact.InlineText); text != "" {
				entry.textOutputs = append(entry.textOutputs, text)
			}
		case strings.HasPrefix(artifact.MimeType, "image/"):
			if entry.imageLink == "" && artifact.AccessURL != "" {
				entry.imageLink = artifact.AccessURL
			}
		case strings.HasPrefix(artifact.MimeType, "video/"):
			if entry.videoLink == "" && artifact.AccessURL != "" {
				entry.videoLink = artifact.AccessURL
			}
		}
	}
	return grouped, nil
}

func localizeBackfillTaskStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed":
		return "成功"
	case "failed", "cancelled":
		return "错误"
	default:
		return strings.TrimSpace(status)
	}
}

func (r *templateBackfillRow) primaryTaskID() string {
	if r == nil {
		return ""
	}
	if r.taskID != "" {
		return r.taskID
	}
	return joinSortedTaskIDs(r.taskIDs)
}

func joinSortedTaskIDs(taskIDs map[string]struct{}) string {
	if len(taskIDs) == 0 {
		return ""
	}
	items := make([]string, 0, len(taskIDs))
	for taskID := range taskIDs {
		items = append(items, taskID)
	}
	sort.Strings(items)
	return strings.Join(items, ",")
}
