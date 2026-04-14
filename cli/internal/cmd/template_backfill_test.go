package cmd

import (
	"context"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

type mockTemplateBackfillQueryClient struct {
	taskPages     []listRunTasksResponse
	pages []listRunArtifactsResponse
	calls int
}

func (m *mockTemplateBackfillQueryClient) GetJSONWithQuery(_ context.Context, path string, query url.Values, out any) error {
	if got := query.Get("pageSize"); got != "200" {
		return &testError{message: "unexpected pageSize: " + got}
	}
	switch resp := out.(type) {
	case *listRunTasksResponse:
		if m.calls >= len(m.taskPages) {
			return &testError{message: "unexpected extra task page request"}
		}
		if path == "" {
			return &testError{message: "missing path"}
		}
		*resp = m.taskPages[m.calls]
	case *listRunArtifactsResponse:
		if m.calls >= len(m.pages) {
			return &testError{message: "unexpected extra artifact page request"}
		}
		if path == "" {
			return &testError{message: "missing path"}
		}
		*resp = m.pages[m.calls]
	default:
		return &testError{message: "unexpected response type"}
	}
	m.calls++
	return nil
}

type testError struct {
	message string
}

func (e *testError) Error() string { return e.message }

func TestListAllArtifacts_Paginates(t *testing.T) {
	client := &mockTemplateBackfillQueryClient{
		pages: []listRunArtifactsResponse{
			{
				Artifacts:     []artifactEntry{{ArtifactID: "a1"}},
				NextPageToken: "next",
			},
			{
				Artifacts: []artifactEntry{{ArtifactID: "a2"}},
			},
		},
	}

	artifacts, err := listAllArtifacts(context.Background(), client, "run-1")
	if err != nil {
		t.Fatalf("listAllArtifacts returned error: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
	if artifacts[0].ArtifactID != "a1" || artifacts[1].ArtifactID != "a2" {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
	if client.calls != 2 {
		t.Fatalf("expected 2 page calls, got %d", client.calls)
	}
}

func TestListAllTasks_Paginates(t *testing.T) {
	client := &mockTemplateBackfillQueryClient{
		taskPages: []listRunTasksResponse{
			{
				Tasks:         []runTaskEntry{{TaskID: "task-1"}},
				NextPageToken: "next",
			},
			{
				Tasks: []runTaskEntry{{TaskID: "task-2"}},
			},
		},
	}

	tasks, err := listAllTasks(context.Background(), client, "run-1")
	if err != nil {
		t.Fatalf("listAllTasks returned error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].TaskID != "task-1" || tasks[1].TaskID != "task-2" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	if client.calls != 2 {
		t.Fatalf("expected 2 page calls, got %d", client.calls)
	}
}

func TestResolveBackfillOutputPath_DefaultsToSameWorkbook(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "filled.xlsx")
	got, err := resolveBackfillOutputPath("", inputPath)
	if err != nil {
		t.Fatalf("resolveBackfillOutputPath returned error: %v", err)
	}
	want, err := filepath.Abs(inputPath)
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected backfill output path: got %q want %q", got, want)
	}
}

func TestGroupTasksAndArtifactsByRow(t *testing.T) {
	tasks := []runTaskEntry{
		{TaskID: "task-1", SourceRowIndex: 0, Status: "failed", ErrorMessage: "bad prompt"},
	}
	artifacts := []artifactEntry{
		{SourceRowIndex: 0, TaskID: "task-2", MimeType: "text/plain", InlineText: "hello"},
		{SourceRowIndex: 0, TaskID: "task-1", MimeType: "image/png", AccessURL: "https://example.com/image.png"},
		{SourceRowIndex: 0, TaskID: "task-1", MimeType: "video/mp4", AccessURL: "https://example.com/video.mp4"},
	}

	grouped, err := groupTasksAndArtifactsByRow(tasks, artifacts, 1)
	if err != nil {
		t.Fatalf("groupTasksAndArtifactsByRow returned error: %v", err)
	}
	row := grouped[0]
	if row == nil {
		t.Fatal("expected row 0 entry")
	}
	if row.status != "failed" || row.errorText != "bad prompt" {
		t.Fatalf("unexpected task state: status=%q error=%q", row.status, row.errorText)
	}
	if got := joinSortedTaskIDs(row.taskIDs); got != "task-1,task-2" {
		t.Fatalf("unexpected task ids: %s", got)
	}
	if row.primaryTaskID() != "task-1" {
		t.Fatalf("unexpected primary task id: %q", row.primaryTaskID())
	}
	if row.imageLink == "" || row.videoLink == "" {
		t.Fatalf("expected media links, got image=%q video=%q", row.imageLink, row.videoLink)
	}
	if len(row.textOutputs) != 1 || row.textOutputs[0] != "hello" {
		t.Fatalf("unexpected text outputs: %+v", row.textOutputs)
	}
}

func TestBackfillWorkbookResults_UsesTaskLevelStatusAndError(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.xlsx")
	outputPath := filepath.Join(dir, "output.xlsx")

	f := excelize.NewFile()
	metaSheet, err := f.NewSheet(templateMetaSheetName)
	if err != nil {
		t.Fatalf("create meta sheet: %v", err)
	}
	dataSheet, err := f.NewSheet(templateDataSheetName)
	if err != nil {
		t.Fatalf("create data sheet: %v", err)
	}
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(dataSheet)
	_ = metaSheet
	if err := f.SetSheetRow(templateMetaSheetName, "A1", &[]string{"__template_id", "text-image-v1"}); err != nil {
		t.Fatalf("set meta row 1: %v", err)
	}
	if err := f.SetSheetRow(templateMetaSheetName, "A2", &[]string{"__template_version", "v1"}); err != nil {
		t.Fatalf("set meta row 2: %v", err)
	}
	if err := f.SetSheetRow(templateDataSheetName, "A1", &[]string{"输入.prompt"}); err != nil {
		t.Fatalf("set data header: %v", err)
	}
	if err := f.SetSheetRow(templateDataSheetName, "A2", &[]string{"row-1"}); err != nil {
		t.Fatalf("set row 1: %v", err)
	}
	if err := f.SetSheetRow(templateDataSheetName, "A3", &[]string{"row-2"}); err != nil {
		t.Fatalf("set row 2: %v", err)
	}
	if err := f.SaveAs(inputPath); err != nil {
		t.Fatalf("save input workbook: %v", err)
	}

	tasks := []runTaskEntry{
		{TaskID: "task-1", SourceRowIndex: 0, Status: "completed"},
		{TaskID: "task-2", SourceRowIndex: 1, Status: "failed", ErrorMessage: "model unavailable"},
	}
	artifacts := []artifactEntry{
		{TaskID: "task-1", SourceRowIndex: 0, MimeType: "text/plain", InlineText: "hello world"},
	}

	rowCount, err := backfillWorkbookResults(inputPath, outputPath, tasks, artifacts)
	if err != nil {
		t.Fatalf("backfillWorkbookResults returned error: %v", err)
	}
	if rowCount != 2 {
		t.Fatalf("expected 2 rows, got %d", rowCount)
	}

	out, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("open output workbook: %v", err)
	}
	defer func() { _ = out.Close() }()

	rows, err := out.GetRows(templateDataSheetName)
	if err != nil {
		t.Fatalf("read output rows: %v", err)
	}
	if got := rows[0][1]; got != "结果.运行状态" {
		t.Fatalf("unexpected status header: %q", got)
	}
	if got := rows[0][2]; got != "结果.错误信息" {
		t.Fatalf("unexpected error header: %q", got)
	}
	if got := rows[1][1]; got != "成功" {
		t.Fatalf("unexpected row 1 status: %q", got)
	}
	if got := rows[1][2]; got != "" {
		t.Fatalf("unexpected row 1 error: %q", got)
	}
	if got := rows[1][3]; got != "hello world" {
		t.Fatalf("unexpected row 1 text output: %q", got)
	}
	if got := rows[1][6]; got != "task-1" {
		t.Fatalf("unexpected row 1 task id: %q", got)
	}
	if got := rows[2][1]; got != "错误" {
		t.Fatalf("unexpected row 2 status: %q", got)
	}
	if got := rows[2][2]; got != "model unavailable" {
		t.Fatalf("unexpected row 2 error: %q", got)
	}
	if got := rows[2][6]; got != "task-2" {
		t.Fatalf("unexpected row 2 task id: %q", got)
	}
}
