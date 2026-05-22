package exporter

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("GITHUB_OWNER", "acme")
	t.Setenv("GITHUB_REPO", "widget")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}

	if cfg.ListenAddress != ":9101" {
		t.Fatalf("ListenAddress = %q, want %q", cfg.ListenAddress, ":9101")
	}
	if cfg.MetricsPath != "/metrics" {
		t.Fatalf("MetricsPath = %q, want %q", cfg.MetricsPath, "/metrics")
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
	if cfg.GitHubReleaseLimit != 0 {
		t.Fatalf("GitHubReleaseLimit = %d, want 0", cfg.GitHubReleaseLimit)
	}
	if cfg.GitHubCacheTTL != 5*time.Minute {
		t.Fatalf("GitHubCacheTTL = %s, want %s", cfg.GitHubCacheTTL, 5*time.Minute)
	}
}

func TestLoadConfigFromEnvOverrides(t *testing.T) {
	t.Setenv("EXPORTER_LISTEN_ADDRESS", ":9999")
	t.Setenv("EXPORTER_METRICS_PATH", "/custom")
	t.Setenv("EXPORTER_LOG_LEVEL", "debug")
	t.Setenv("GITHUB_OWNER", "acme")
	t.Setenv("GITHUB_REPO", "widget")
	t.Setenv("GITHUB_TOKEN", "token-123")
	t.Setenv("GITHUB_RELEASE_LIMIT", "7")
	t.Setenv("GITHUB_CACHE_TTL", "30s")
	t.Setenv("GITHUB_HTTP_TIMEOUT", "4s")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}

	if cfg.ListenAddress != ":9999" {
		t.Fatalf("ListenAddress = %q, want %q", cfg.ListenAddress, ":9999")
	}
	if cfg.MetricsPath != "/custom" {
		t.Fatalf("MetricsPath = %q, want %q", cfg.MetricsPath, "/custom")
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
	if cfg.GitHubToken != "token-123" {
		t.Fatalf("GitHubToken = %q, want %q", cfg.GitHubToken, "token-123")
	}
	if cfg.GitHubReleaseLimit != 7 {
		t.Fatalf("GitHubReleaseLimit = %d, want 7", cfg.GitHubReleaseLimit)
	}
	if cfg.GitHubCacheTTL != 30*time.Second {
		t.Fatalf("GitHubCacheTTL = %s, want %s", cfg.GitHubCacheTTL, 30*time.Second)
	}
	if cfg.GitHubHTTPTimeout != 4*time.Second {
		t.Fatalf("GitHubHTTPTimeout = %s, want %s", cfg.GitHubHTTPTimeout, 4*time.Second)
	}
}

func TestLoadConfigFromEnvValidation(t *testing.T) {
	t.Setenv("GITHUB_OWNER", "")
	t.Setenv("GITHUB_REPO", "")

	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("expected error for missing required env vars")
	}

	t.Setenv("GITHUB_OWNER", "acme")
	t.Setenv("GITHUB_REPO", "widget")
	t.Setenv("GITHUB_RELEASE_LIMIT", "-1")
	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("expected error for negative GITHUB_RELEASE_LIMIT")
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    slog.Level
		wantErr bool
	}{
		{name: "debug", input: "debug", want: slog.LevelDebug},
		{name: "info", input: "INFO", want: slog.LevelInfo},
		{name: "warn", input: "warning", want: slog.LevelWarn},
		{name: "error", input: "error", want: slog.LevelError},
		{name: "invalid", input: "verbose", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLogLevel(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLogLevel() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("parseLogLevel() = %v, want %v", got, tc.want)
			}
		})
	}
}
