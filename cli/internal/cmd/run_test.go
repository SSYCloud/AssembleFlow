package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunResultRowsCmdUsesSnapshotResultEndpoint(t *testing.T) {
	var requestedPath string
	var requestedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"rows": [
				{
					"rowIndex": 0,
					"status": "completed",
					"inputJson": "{\"prompt\":\"hello\"}",
					"artifacts": [{"artifactId":"art_1","mimeType":"text/plain"}]
				}
			],
			"nextPageToken": "2",
			"totalCount": 3
		}`))
	}))
	defer server.Close()

	opts := &rootOptions{
		server:  server.URL,
		timeout: time.Second,
		output:  "text",
	}
	cmd := newRunResultRowsCmd(opts)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"run_123", "--page-size", "2", "--page-token", "1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("result-rows command error = %v", err)
	}
	if requestedPath != "/v1/batch/workflow-runs/run_123/result-rows" {
		t.Fatalf("path=%q want result-rows endpoint", requestedPath)
	}
	for _, want := range []string{"pageSize=2", "pageToken=1"} {
		if !strings.Contains(requestedQuery, want) {
			t.Fatalf("query %q missing %q", requestedQuery, want)
		}
	}
	if !strings.Contains(out.String(), "completed") || !strings.Contains(out.String(), "total_count\t3") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestRunResultWorkbookCmdDownloadsServerWorkbook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/batch/workflow-runs/run_123/result-workbook" {
			t.Fatalf("path=%q want result-workbook endpoint", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", `attachment; filename="result-run_123.xlsx"`)
		_, _ = w.Write([]byte("xlsx bytes"))
	}))
	defer server.Close()

	outDir := t.TempDir()
	opts := &rootOptions{
		server:  server.URL,
		timeout: time.Second,
		output:  "json",
	}
	cmd := newRunResultWorkbookCmd(opts)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"run_123", "--output-file", outDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("result-workbook command error = %v", err)
	}
	target := filepath.Join(outDir, "result-run_123.xlsx")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read result workbook: %v", err)
	}
	if string(data) != "xlsx bytes" {
		t.Fatalf("downloaded bytes=%q", string(data))
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode output JSON: %v", err)
	}
	if payload["path"] != target {
		t.Fatalf("output path=%v want %s", payload["path"], target)
	}
}
