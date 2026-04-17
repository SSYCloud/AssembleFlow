package version

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
	}{
		{name: "older", current: "v0.1.4", latest: "v0.1.5", want: -1},
		{name: "equal", current: "v0.1.5", latest: "v0.1.5", want: 0},
		{name: "newer", current: "v0.1.6", latest: "v0.1.5", want: 1},
		{name: "dev fallback", current: "dev", latest: "v0.1.5", want: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareVersions(tt.current, tt.latest); got != tt.want {
				t.Fatalf("compareVersions(%q, %q)=%d want %d", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestCheckLatest(t *testing.T) {
	origVersion := Version
	t.Cleanup(func() {
		Version = origVersion
		_ = os.Unsetenv("LOOMLOOM_CLI_RELEASE_API")
		_ = os.Unsetenv("BATCHJOB_CLI_RELEASE_API")
	})
	Version = "v0.1.4"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.1.5"}`))
	}))
	defer server.Close()
	_ = os.Setenv("LOOMLOOM_CLI_RELEASE_API", server.URL)

	status, err := CheckLatest(context.Background())
	if err != nil {
		t.Fatalf("CheckLatest() error = %v", err)
	}
	if status.CurrentVersion != "v0.1.4" {
		t.Fatalf("CurrentVersion=%q", status.CurrentVersion)
	}
	if status.LatestVersion != "v0.1.5" {
		t.Fatalf("LatestVersion=%q", status.LatestVersion)
	}
	if !status.UpdateAvailable {
		t.Fatalf("UpdateAvailable=false, want true")
	}
	if status.UpgradeHint == "" {
		t.Fatalf("UpgradeHint should not be empty")
	}
}
