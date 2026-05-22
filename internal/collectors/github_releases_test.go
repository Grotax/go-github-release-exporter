package collectors

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoadReleaseAssetsUsesCache(t *testing.T) {
	var hitCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"name":"Release 1","tag_name":"v1.0.0","assets":[{"name":"artifact-a","download_count":42}]}
		]`)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewGitHubReleasesCollector(GitHubReleasesCollectorConfig{
		Owner:       "acme",
		Repo:        "widget",
		CacheTTL:    5 * time.Minute,
		HTTPTimeout: 3 * time.Second,
		APIBaseURL:  ts.URL,
	}, logger)

	first, err := collector.loadReleaseAssets()
	if err != nil {
		t.Fatalf("first loadReleaseAssets() error = %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("first loadReleaseAssets() len = %d, want 1", len(first))
	}
	if first[0].DownloadCount != 42 {
		t.Fatalf("download count = %v, want 42", first[0].DownloadCount)
	}

	second, err := collector.loadReleaseAssets()
	if err != nil {
		t.Fatalf("second loadReleaseAssets() error = %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("second loadReleaseAssets() len = %d, want 1", len(second))
	}

	if got := atomic.LoadInt32(&hitCount); got != 1 {
		t.Fatalf("api hit count = %d, want 1 (cached second call)", got)
	}
}

func TestFetchReleasesRespectsLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"name":"Release 2","tag_name":"v2.0.0","assets":[{"name":"artifact-b","download_count":9}]},
			{"name":"Release 1","tag_name":"v1.0.0","assets":[{"name":"artifact-a","download_count":7}]}
		]`)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewGitHubReleasesCollector(GitHubReleasesCollectorConfig{
		Owner:        "acme",
		Repo:         "widget",
		ReleaseLimit: 1,
		CacheTTL:     5 * time.Minute,
		HTTPTimeout:  3 * time.Second,
		APIBaseURL:   ts.URL,
	}, logger)

	releases, err := collector.fetchReleases(context.Background())
	if err != nil {
		t.Fatalf("fetchReleases() error = %v", err)
	}
	if len(releases) != 1 {
		t.Fatalf("fetchReleases() len = %d, want 1", len(releases))
	}
	if releases[0].TagName != "v2.0.0" {
		t.Fatalf("first release tag = %q, want %q", releases[0].TagName, "v2.0.0")
	}
}

func TestLoadReleaseAssetsErrorFromGitHub(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	collector := NewGitHubReleasesCollector(GitHubReleasesCollectorConfig{
		Owner:       "acme",
		Repo:        "widget",
		CacheTTL:    5 * time.Minute,
		HTTPTimeout: 3 * time.Second,
		APIBaseURL:  ts.URL,
	}, logger)

	if _, err := collector.loadReleaseAssets(); err == nil {
		t.Fatal("expected error from loadReleaseAssets(), got nil")
	}
}
