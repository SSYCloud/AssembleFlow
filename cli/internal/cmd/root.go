package cmd

import (
	"os"
	"time"

	"github.com/SSYCloud/batchjob-agent-kit/cli/internal/client"
	"github.com/SSYCloud/batchjob-agent-kit/cli/internal/version"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	server  string
	token   string
	timeout time.Duration
	output  string
}

func NewRootCmd() *cobra.Command {
	opts := &rootOptions{
		server:  envOrDefault("BATCHJOB_SERVER", "http://127.0.0.1:8080"),
		token:   os.Getenv("BATCHJOB_TOKEN"),
		timeout: 30 * time.Second,
		output:  "text",
	}

	cmd := &cobra.Command{
		Use:           "batchjob-cli",
		Short:         "Developer CLI for hosted BatchJob skills",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
	}

	cmd.PersistentFlags().StringVarP(&opts.server, "server", "s", opts.server, "BatchJob base URL or host")
	cmd.PersistentFlags().StringVarP(&opts.token, "token", "t", opts.token, "Bearer token")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", opts.timeout, "HTTP timeout")
	cmd.PersistentFlags().StringVarP(&opts.output, "output", "o", opts.output, "Output format: text|json")

	cmd.AddCommand(
		newDoctorCmd(opts),
		newInputAssetCmd(opts),
		newModelCmd(opts),
		newRunCmd(opts),
		newTemplateCmd(opts),
		newArtifactCmd(opts),
	)
	return cmd
}

func newHTTPClient(opts *rootOptions) (*client.Client, error) {
	return client.New(client.Config{
		BaseURL: opts.server,
		Token:   opts.token,
		Timeout: opts.timeout,
	})
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
