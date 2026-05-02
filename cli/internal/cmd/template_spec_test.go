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

func TestLoadTemplateSpecFile_ValidSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.json")
	content := `{
  "Meta": {"Name": "Spec Test", "Description": "desc"},
  "Steps": [{"StepID": "stp_text", "DisplayName": "Text", "ExecutionUnit": "text-generate"}],
  "InputSchema": {"Fields": [{"Key": "prompt", "Label": "Prompt", "ValueType": "string"}]},
  "FieldBindings": [{"FieldKey": "prompt", "StepID": "stp_text", "ParamKey": "prompt", "BindMode": "shared"}]
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	spec, raw, err := loadTemplateSpecFile(path)
	if err != nil {
		t.Fatalf("loadTemplateSpecFile() error = %v", err)
	}
	if spec.Meta.Name != "Spec Test" {
		t.Fatalf("Meta.Name = %q, want Spec Test", spec.Meta.Name)
	}
	if len(raw) == 0 || raw[0] != '{' {
		t.Fatalf("expected compact JSON bytes, got %q", string(raw))
	}
	if !strings.Contains(string(raw), `"meta"`) {
		t.Fatalf("expected normalized lowerCamel TemplateSpec JSON, got %s", string(raw))
	}
	if strings.Contains(string(raw), `"Meta"`) {
		t.Fatalf("expected PascalCase keys to be normalized, got %s", string(raw))
	}
}

func TestLoadTemplateSpecFile_MissingName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.json")
	content := `{
  "Meta": {},
  "Steps": [{"StepID": "stp_text"}],
  "InputSchema": {"Fields": []},
  "FieldBindings": []
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	if _, _, err := loadTemplateSpecFile(path); err == nil {
		t.Fatal("loadTemplateSpecFile() error = nil, want missing name error")
	}
}

func TestTemplateSpecModelsCmdListsAvailableModels(t *testing.T) {
	var requestedPath string
	var requestedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"models": [
				{
					"modelId": "google/gemini-2.5-flash",
					"displayName": "Gemini 2.5 Flash",
					"provider": "vertex",
					"executionAdapter": "vertex",
					"supportedStepTypes": ["text-generate"],
					"available": true,
					"isDefault": true
				}
			]
		}`))
	}))
	defer server.Close()

	opts := &rootOptions{
		server:  server.URL,
		timeout: time.Second,
		output:  "text",
	}
	cmd := newTemplateSpecModelsCmd(opts)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"text-generate"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models command error = %v", err)
	}
	if requestedPath != "/v1/batch/models" {
		t.Fatalf("path=%q want /v1/batch/models", requestedPath)
	}
	for _, want := range []string{"stepType=text-generate", "provider=vertex", "onlyAvailable=true"} {
		if !strings.Contains(requestedQuery, want) {
			t.Fatalf("query %q missing %q", requestedQuery, want)
		}
	}
	if !strings.Contains(out.String(), "google/gemini-2.5-flash") {
		t.Fatalf("output missing model id: %s", out.String())
	}
}

func TestTemplateSpecModelsCmdCanIncludeUnavailableModels(t *testing.T) {
	var requestedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer server.Close()

	opts := &rootOptions{
		server:  server.URL,
		timeout: time.Second,
		output:  "json",
	}
	cmd := newTemplateSpecModelsCmd(opts)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"image-generate", "--include-unavailable"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("models command error = %v", err)
	}
	if !strings.Contains(requestedQuery, "onlyAvailable=false") {
		t.Fatalf("query %q missing onlyAvailable=false", requestedQuery)
	}
	if !strings.Contains(out.String(), `"models": []`) {
		t.Fatalf("json output missing models array: %s", out.String())
	}
}

func TestTemplateSpecSubmitWorkbookSendsFilename(t *testing.T) {
	workbookPath := filepath.Join(t.TempDir(), "custom-input.xlsx")
	if err := os.WriteFile(workbookPath, []byte("xlsx bytes"), 0o644); err != nil {
		t.Fatalf("write workbook: %v", err)
	}

	var submitFilename string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		switch r.URL.Path {
		case "/v1/batch/user-template-workbook:validate":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"valid":true}`))
		case "/v1/batch/user-template-workbook:submit":
			submitFilename, _ = payload["filename"].(string)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"runId":"run_123","status":"pending","acceptedAt":"1777699967"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	opts := &rootOptions{
		server:  server.URL,
		timeout: time.Second,
		output:  "json",
	}
	cmd := newTemplateSpecSubmitWorkbookCmd(opts)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"tmpl_123", "ver_123", workbookPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("submit-workbook command error = %v", err)
	}
	if submitFilename != "custom-input.xlsx" {
		t.Fatalf("filename=%q want custom-input.xlsx", submitFilename)
	}
	if !strings.Contains(out.String(), `"runId": "run_123"`) {
		t.Fatalf("output missing run id: %s", out.String())
	}
}
