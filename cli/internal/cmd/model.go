package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

type listModelsResponse struct {
	Models []modelSummary `json:"models"`
}

type getModelResponse struct {
	Model *modelDetail `json:"model"`
}

type modelSummary struct {
	ModelID            string   `json:"modelId"`
	DisplayName        string   `json:"displayName"`
	Provider           string   `json:"provider"`
	SupportedStepTypes []string `json:"supportedStepTypes"`
	InputModalities    []string `json:"inputModalities"`
	OutputModalities   []string `json:"outputModalities"`
	Available          bool     `json:"available"`
	AvailabilityReason string   `json:"availabilityReason"`
}

type modelDetail struct {
	ModelID            string   `json:"modelId"`
	DisplayName        string   `json:"displayName"`
	Provider           string   `json:"provider"`
	SupportedStepTypes []string `json:"supportedStepTypes"`
	InputModalities    []string `json:"inputModalities"`
	OutputModalities   []string `json:"outputModalities"`
	SupportedAPIs      []string `json:"supportedApis"`
	Available          bool     `json:"available"`
	AvailabilityReason string   `json:"availabilityReason"`
}

func newModelCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Discover executable models by step type",
	}

	cmd.AddCommand(
		newModelListCmd(opts),
		newModelGetCmd(opts),
	)
	return cmd
}

func newModelListCmd(opts *rootOptions) *cobra.Command {
	var stepType string
	var provider string
	var onlyAvailable bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List executable models for one step type",
		RunE: func(cmd *cobra.Command, args []string) error {
			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			query := url.Values{}
			query.Set("step_type", strings.TrimSpace(stepType))
			if strings.TrimSpace(provider) != "" {
				query.Set("provider", strings.TrimSpace(provider))
			}
			query.Set("only_available", fmt.Sprintf("%t", onlyAvailable))

			var resp listModelsResponse
			if err := httpClient.GetJSONWithQuery(ctx, "/v1/batch/models", query, &resp); err != nil {
				return err
			}

			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			if len(resp.Models) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no models")
				return nil
			}

			return printModelSummaries(cmd.OutOrStdout(), resp.Models)
		},
	}

	cmd.Flags().StringVar(&stepType, "step-type", "", "Required step type: text-generate|image-generate|video-generate")
	cmd.Flags().StringVar(&provider, "provider", "vertex", "Optional provider filter")
	cmd.Flags().BoolVar(&onlyAvailable, "only-available", true, "Only show currently executable models")
	_ = cmd.MarkFlagRequired("step-type")
	return cmd
}

func newModelGetCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <model-id>",
		Short: "Inspect one executable model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			httpClient, err := newHTTPClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), opts.timeout)
			defer cancel()

			path := "/v1/batch/model-details/" + url.PathEscape(strings.TrimSpace(args[0]))
			var resp getModelResponse
			if err := httpClient.GetJSON(ctx, path, &resp); err != nil {
				return err
			}

			if opts.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			if resp.Model == nil {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "model not found")
				return nil
			}
			return printModelDetail(cmd.OutOrStdout(), *resp.Model)
		},
	}
}
