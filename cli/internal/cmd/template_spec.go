package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type createUserTemplateResponse struct {
	TemplateID  string    `json:"templateId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   flexInt64 `json:"createdAt"`
}

type saveTemplateVersionResponse struct {
	VersionID      string    `json:"versionId"`
	VersionNumber  flexInt64 `json:"versionNumber"`
	DefinitionHash string    `json:"definitionHash"`
	CreatedAt      flexInt64 `json:"createdAt"`
}

type downloadUserTemplateWorkbookResponse struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
}

type submitUserTemplateWorkbookResponse struct {
	RunID      string    `json:"runId"`
	Status     string    `json:"status"`
	AcceptedAt flexInt64 `json:"acceptedAt"`
}

type listModelsResponse struct {
	Models []modelSummary `json:"models"`
}

type modelSummary struct {
	ModelID            string   `json:"modelId"`
	DisplayName        string   `json:"displayName"`
	Provider           string   `json:"provider"`
	ExecutionAdapter   string   `json:"executionAdapter"`
	SupportedStepTypes []string `json:"supportedStepTypes"`
	SupportedAPIs      []string `json:"supportedApis"`
	Available          bool     `json:"available"`
	AvailabilityReason string   `json:"availabilityReason"`
	IsDefault          bool     `json:"isDefault"`
}

type templateSpecMeta struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
}

type templateSpecEnvelope struct {
	Meta          templateSpecMeta `json:"Meta"`
	Steps         []any            `json:"Steps"`
	InputSchema   any              `json:"InputSchema"`
	FieldBindings []any            `json:"FieldBindings"`
}

func newTemplateSpecCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template-spec",
		Short: "Author user templates from TemplateSpec JSON",
	}
	cmd.AddCommand(
		newTemplateSpecCheckCmd(opts),
		newTemplateSpecModelsCmd(opts),
		newTemplateSpecCreateCmd(opts),
		newTemplateSpecDownloadWorkbookCmd(opts),
		newTemplateSpecValidateWorkbookCmd(opts),
		newTemplateSpecSubmitWorkbookCmd(opts),
	)
	return cmd
}

func newTemplateSpecCheckCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "check <spec-json>",
		Short: "Check that a TemplateSpec JSON file is parseable and has the required top-level shape",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, raw, err := loadTemplateSpecFile(args[0])
			if err != nil {
				return err
			}
			result := map[string]any{
				"valid":       true,
				"name":        spec.Meta.Name,
				"description": spec.Meta.Description,
				"steps":       len(spec.Steps),
				"bindings":    len(spec.FieldBindings),
				"bytes":       len(raw),
			}
			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"valid\nname\t%s\nsteps\t%d\nbindings\t%d\nbytes\t%d\n",
				spec.Meta.Name,
				len(spec.Steps),
				len(spec.FieldBindings),
				len(raw),
			)
			return err
		},
	}
}

func newTemplateSpecModelsCmd(opts *rootOptions) *cobra.Command {
	var provider string
	var includeUnavailable bool

	cmd := &cobra.Command{
		Use:   "models <step-type>",
		Short: "List executable models available for a TemplateSpec step type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stepType := strings.TrimSpace(args[0])
			if stepType == "" {
				return errors.New("step-type is required")
			}
			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			query := url.Values{}
			query.Set("stepType", stepType)
			query.Set("onlyAvailable", fmt.Sprintf("%t", !includeUnavailable))
			if strings.TrimSpace(provider) != "" {
				query.Set("provider", strings.TrimSpace(provider))
			}

			var resp listModelsResponse
			if err := httpClient.GetJSONWithQuery(ctx, "/v1/batch/models", query, &resp); err != nil {
				return err
			}
			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}
			return printTemplateSpecModels(cmd.OutOrStdout(), resp.Models)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "vertex", "Model provider filter")
	cmd.Flags().BoolVar(&includeUnavailable, "include-unavailable", false, "Include known but currently unavailable models")
	return cmd
}

func newTemplateSpecCreateCmd(opts *rootOptions) *cobra.Command {
	var name string
	var description string
	var versionNote string

	cmd := &cobra.Command{
		Use:   "create <spec-json>",
		Short: "Create a private user template and save the TemplateSpec JSON as version 1",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, raw, err := loadTemplateSpecFile(args[0])
			if err != nil {
				return err
			}
			effectiveName := firstNonEmpty(name, spec.Meta.Name)
			if effectiveName == "" {
				return errors.New("template name is required; set Meta.Name or pass --name")
			}
			effectiveDescription := firstNonEmpty(description, spec.Meta.Description)

			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			var createResp createUserTemplateResponse
			if err := httpClient.PostJSON(ctx, "/v1/user-templates", map[string]any{
				"name":        effectiveName,
				"description": effectiveDescription,
			}, &createResp); err != nil {
				return err
			}

			var versionResp saveTemplateVersionResponse
			if err := httpClient.PostJSON(ctx, "/v1/user-templates/"+createResp.TemplateID+"/versions", map[string]any{
				"versionNote":       strings.TrimSpace(versionNote),
				"canonicalSpecJson": string(raw),
			}, &versionResp); err != nil {
				_ = httpClient.PostJSON(ctx, "/v1/user-templates/"+createResp.TemplateID+":archive", map[string]any{}, nil)
				return fmt.Errorf("save template version for %s: %w", createResp.TemplateID, err)
			}

			result := map[string]any{
				"templateId":     createResp.TemplateID,
				"name":           createResp.Name,
				"description":    createResp.Description,
				"status":         createResp.Status,
				"versionId":      versionResp.VersionID,
				"versionNumber":  int64(versionResp.VersionNumber),
				"definitionHash": versionResp.DefinitionHash,
			}
			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"template_id\t%s\nname\t%s\nversion_id\t%s\nversion_number\t%d\ndefinition_hash\t%s\n",
				createResp.TemplateID,
				createResp.Name,
				versionResp.VersionID,
				int64(versionResp.VersionNumber),
				versionResp.DefinitionHash,
			)
			return err
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Template name override; defaults to Meta.Name")
	cmd.Flags().StringVar(&description, "description", "", "Template description override; defaults to Meta.Description")
	cmd.Flags().StringVar(&versionNote, "version-note", "", "Optional note for version 1")
	return cmd
}

func newTemplateSpecDownloadWorkbookCmd(opts *rootOptions) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "download-workbook <template-id> <version-id>",
		Short: "Download the Excel workbook generated from a user template version",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			templateID := strings.TrimSpace(args[0])
			versionID := strings.TrimSpace(args[1])
			var resp downloadUserTemplateWorkbookResponse
			if err := httpClient.GetJSON(ctx, "/v1/user-templates/"+templateID+"/versions/"+versionID+"/workbook", &resp); err != nil {
				return err
			}
			filename := resp.Filename
			if filename == "" {
				filename = templateID + "-" + versionID + ".xlsx"
			}
			targetPath, err := resolveFilePath(outputPath, filepath.Base(filename))
			if err != nil {
				return fmt.Errorf("resolve output file path: %w", err)
			}
			if err := os.WriteFile(targetPath, resp.Content, 0o644); err != nil {
				return fmt.Errorf("write downloaded file: %w", err)
			}
			result := map[string]any{
				"templateId": templateID,
				"versionId":  versionID,
				"path":       targetPath,
				"filename":   filename,
				"size":       len(resp.Content),
			}
			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "template_id\t%s\nversion_id\t%s\npath\t%s\nsize\t%d\n", templateID, versionID, targetPath, len(resp.Content))
			return err
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output-file", "f", "", "Output .xlsx path or target directory")
	return cmd
}

func newTemplateSpecValidateWorkbookCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "validate-workbook <template-id> <version-id> <xlsx-path>",
		Short: "Validate a filled user-template workbook",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := postUserTemplateWorkbook[validateTemplateFileResponse](cmd.Context(), opts, args[0], args[1], args[2], "/v1/batch/user-template-workbook:validate", nil)
			if err != nil {
				return err
			}
			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"templateId": args[0],
					"versionId":  args[1],
					"file":       args[2],
					"validation": resp,
				})
			}
			if err := printTemplateFileValidation(cmd.OutOrStdout(), resp); err != nil {
				return err
			}
			if !resp.Valid {
				return templateFileValidationError(resp)
			}
			return nil
		},
	}
}

func newTemplateSpecSubmitWorkbookCmd(opts *rootOptions) *cobra.Command {
	var callbackURL string
	var idempotencyKey string

	cmd := &cobra.Command{
		Use:   "submit-workbook <template-id> <version-id> <xlsx-path>",
		Short: "Submit a filled user-template workbook as a run",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			validateResp, err := postUserTemplateWorkbook[validateTemplateFileResponse](cmd.Context(), opts, args[0], args[1], args[2], "/v1/batch/user-template-workbook:validate", nil)
			if err != nil {
				return err
			}
			if !validateResp.Valid {
				if opts.output == "json" {
					enc := json.NewEncoder(cmd.OutOrStdout())
					enc.SetIndent("", "  ")
					_ = enc.Encode(map[string]any{
						"templateId": args[0],
						"versionId":  args[1],
						"file":       args[2],
						"validation": validateResp,
					})
				}
				return templateFileValidationError(validateResp)
			}

			extra := map[string]string{}
			if strings.TrimSpace(callbackURL) != "" {
				extra["callbackUrl"] = strings.TrimSpace(callbackURL)
			}
			if strings.TrimSpace(idempotencyKey) != "" {
				extra["idempotencyKey"] = strings.TrimSpace(idempotencyKey)
			}
			submitResp, err := postUserTemplateWorkbook[submitUserTemplateWorkbookResponse](cmd.Context(), opts, args[0], args[1], args[2], "/v1/batch/user-template-workbook:submit", extra)
			if err != nil {
				return err
			}
			result := map[string]any{
				"templateId": args[0],
				"versionId":  args[1],
				"file":       args[2],
				"runId":      submitResp.RunID,
				"status":     submitResp.Status,
				"acceptedAt": int64(submitResp.AcceptedAt),
			}
			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"template_id\t%s\nversion_id\t%s\nfile\t%s\nrun_id\t%s\nstatus\t%s\naccepted_at\t%s\n",
				args[0],
				args[1],
				args[2],
				submitResp.RunID,
				submitResp.Status,
				formatUnix(int64(submitResp.AcceptedAt)),
			)
			return err
		},
	}
	cmd.Flags().StringVar(&callbackURL, "callback-url", "", "Optional callback URL")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Optional idempotency key")
	return cmd
}

func loadTemplateSpecFile(path string) (templateSpecEnvelope, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return templateSpecEnvelope{}, nil, fmt.Errorf("read %s: %w", path, err)
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return templateSpecEnvelope{}, nil, errors.New("template spec file is empty")
	}
	var spec templateSpecEnvelope
	if err := json.Unmarshal(trimmed, &spec); err != nil {
		return templateSpecEnvelope{}, nil, fmt.Errorf("parse TemplateSpec JSON: %w", err)
	}
	if strings.TrimSpace(spec.Meta.Name) == "" {
		return templateSpecEnvelope{}, nil, errors.New("TemplateSpec Meta.Name is required")
	}
	if len(spec.Steps) == 0 {
		return templateSpecEnvelope{}, nil, errors.New("TemplateSpec Steps must not be empty")
	}
	if spec.InputSchema == nil {
		return templateSpecEnvelope{}, nil, errors.New("TemplateSpec InputSchema is required")
	}
	if spec.FieldBindings == nil {
		return templateSpecEnvelope{}, nil, errors.New("TemplateSpec FieldBindings is required")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err != nil {
		return templateSpecEnvelope{}, nil, fmt.Errorf("compact TemplateSpec JSON: %w", err)
	}
	return spec, compact.Bytes(), nil
}

func postUserTemplateWorkbook[T any](ctx context.Context, opts *rootOptions, templateID, versionID, workbookPath, endpoint string, extra map[string]string) (T, error) {
	var zero T
	content, err := os.ReadFile(workbookPath)
	if err != nil {
		return zero, fmt.Errorf("read workbook: %w", err)
	}
	httpClient, err := newHTTPClient(opts)
	if err != nil {
		return zero, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	payload := map[string]any{
		"templateId": strings.TrimSpace(templateID),
		"versionId":  strings.TrimSpace(versionID),
		"content":    content,
	}
	for key, value := range extra {
		payload[key] = value
	}
	var out T
	if err := httpClient.PostJSON(requestCtx, endpoint, payload, &out); err != nil {
		return zero, err
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func printTemplateSpecModels(w interface {
	Write([]byte) (int, error)
}, models []modelSummary) error {
	if len(models) == 0 {
		_, err := fmt.Fprintln(w, "no models")
		return err
	}
	tw := newTabWriter(w)
	if _, err := fmt.Fprintln(tw, "model_id\tdisplay_name\tprovider\tdefault\tavailable\treason"); err != nil {
		return err
	}
	for _, model := range models {
		reason := model.AvailabilityReason
		if reason == "" {
			reason = "-"
		}
		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%t\t%t\t%s\n",
			model.ModelID,
			model.DisplayName,
			model.Provider,
			model.IsDefault,
			model.Available,
			reason,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}
