package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	// Version is injected at release build time. Local builds keep the default.
	Version = "dev"
)

const (
	repoOwnerRepo      = "SSYCloud/batchjob-agent-kit"
	defaultReleaseAPI  = "https://api.github.com/repos/" + repoOwnerRepo + "/releases/latest"
	defaultHTTPTimeout = 5 * time.Second
)

type Status struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	UpgradeHint     string
}

type latestReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func CheckLatest(ctx context.Context) (*Status, error) {
	status := &Status{CurrentVersion: strings.TrimSpace(Version)}
	if status.CurrentVersion == "" {
		status.CurrentVersion = "dev"
	}

	latest, err := fetchLatestVersion(ctx, releaseAPIURL())
	if err != nil {
		return status, err
	}
	status.LatestVersion = latest

	switch compareVersions(status.CurrentVersion, latest) {
	case -1:
		status.UpdateAvailable = true
		status.UpgradeHint = fmt.Sprintf(
			"new CLI release available: %s (current: %s). Run the installer again to update CLI and skill together.",
			latest, status.CurrentVersion,
		)
	default:
		status.UpdateAvailable = false
	}

	return status, nil
}

func releaseAPIURL() string {
	if v := strings.TrimSpace(os.Getenv("BATCHJOB_CLI_RELEASE_API")); v != "" {
		return v
	}
	return defaultReleaseAPI
}

func fetchLatestVersion(ctx context.Context, apiURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "batchjob-cli/"+strings.TrimSpace(Version))

	httpClient := &http.Client{Timeout: defaultHTTPTimeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("check latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("check latest release: unexpected status %d", resp.StatusCode)
	}

	var payload latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode latest release response: %w", err)
	}
	tag := strings.TrimSpace(payload.TagName)
	if tag == "" {
		return "", fmt.Errorf("latest release response missing tag_name")
	}
	return tag, nil
}

func compareVersions(current, latest string) int {
	cur, okCur := parseVersion(current)
	lat, okLat := parseVersion(latest)
	if !okCur || !okLat {
		return strings.Compare(current, latest)
	}
	for i := 0; i < len(cur) && i < len(lat); i++ {
		if cur[i] < lat[i] {
			return -1
		}
		if cur[i] > lat[i] {
			return 1
		}
	}
	if len(cur) < len(lat) {
		return -1
	}
	if len(cur) > len(lat) {
		return 1
	}
	return 0
}

func parseVersion(raw string) ([]int, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")
	if raw == "" {
		return nil, false
	}
	parts := strings.Split(raw, ".")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
		num, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		out = append(out, num)
	}
	return out, true
}
