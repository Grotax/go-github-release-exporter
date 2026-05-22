package exporter

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains runtime configuration for the exporter.
type Config struct {
	ListenAddress      string
	MetricsPath        string
	LogLevel           slog.Level
	GitHubToken        string
	GitHubOwner        string
	GitHubRepo         string
	GitHubReleaseLimit int
	GitHubCacheTTL     time.Duration
	GitHubHTTPTimeout  time.Duration
}

// DefaultConfig returns a minimal default configuration.
func DefaultConfig() Config {
	return Config{
		ListenAddress:      ":9101",
		MetricsPath:        "/metrics",
		LogLevel:           slog.LevelInfo,
		GitHubReleaseLimit: 0,
		GitHubCacheTTL:     5 * time.Minute,
		GitHubHTTPTimeout:  15 * time.Second,
	}
}

// LoadConfigFromEnv loads exporter configuration from environment variables.
func LoadConfigFromEnv() (Config, error) {
	cfg := DefaultConfig()

	if v := strings.TrimSpace(os.Getenv("EXPORTER_LISTEN_ADDRESS")); v != "" {
		cfg.ListenAddress = v
	}

	if v := strings.TrimSpace(os.Getenv("EXPORTER_METRICS_PATH")); v != "" {
		cfg.MetricsPath = v
	}

	if v := strings.TrimSpace(os.Getenv("EXPORTER_LOG_LEVEL")); v != "" {
		level, err := parseLogLevel(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse EXPORTER_LOG_LEVEL: %w", err)
		}
		cfg.LogLevel = level
	}

	cfg.GitHubToken = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	cfg.GitHubOwner = strings.TrimSpace(os.Getenv("GITHUB_OWNER"))
	cfg.GitHubRepo = strings.TrimSpace(os.Getenv("GITHUB_REPO"))

	if cfg.GitHubOwner == "" {
		return Config{}, fmt.Errorf("GITHUB_OWNER is required")
	}

	if cfg.GitHubRepo == "" {
		return Config{}, fmt.Errorf("GITHUB_REPO is required")
	}

	if v := strings.TrimSpace(os.Getenv("GITHUB_RELEASE_LIMIT")); v != "" {
		releaseLimit, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse GITHUB_RELEASE_LIMIT: %w", err)
		}
		if releaseLimit < 0 {
			return Config{}, fmt.Errorf("GITHUB_RELEASE_LIMIT must be >= 0")
		}
		cfg.GitHubReleaseLimit = releaseLimit
	}

	if v := strings.TrimSpace(os.Getenv("GITHUB_CACHE_TTL")); v != "" {
		cacheTTL, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse GITHUB_CACHE_TTL: %w", err)
		}
		if cacheTTL <= 0 {
			return Config{}, fmt.Errorf("GITHUB_CACHE_TTL must be > 0")
		}
		cfg.GitHubCacheTTL = cacheTTL
	}

	if v := strings.TrimSpace(os.Getenv("GITHUB_HTTP_TIMEOUT")); v != "" {
		httpTimeout, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("parse GITHUB_HTTP_TIMEOUT: %w", err)
		}
		if httpTimeout <= 0 {
			return Config{}, fmt.Errorf("GITHUB_HTTP_TIMEOUT must be > 0")
		}
		cfg.GitHubHTTPTimeout = httpTimeout
	}

	return cfg, nil
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unsupported level %q", raw)
	}
}
