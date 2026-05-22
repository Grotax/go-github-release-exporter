package main

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/grotax/go-github-release-exporter/internal/collectors"
	"github.com/grotax/go-github-release-exporter/internal/exporter"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := exporter.LoadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	logger := exporter.NewLogger(cfg.LogLevel)

	srv := exporter.NewServer(cfg, logger)

	if err := srv.RegisterCollector(collectors.NewGitHubReleasesCollector(collectors.GitHubReleasesCollectorConfig{
		Owner:        cfg.GitHubOwner,
		Repo:         cfg.GitHubRepo,
		Token:        cfg.GitHubToken,
		ReleaseLimit: cfg.GitHubReleaseLimit,
		CacheTTL:     cfg.GitHubCacheTTL,
		HTTPTimeout:  cfg.GitHubHTTPTimeout,
	}, logger)); err != nil {
		logger.Error("failed to register collector", slog.Any("error", err))
		panic(fmt.Errorf("register collector: %w", err))
	}

	if err := srv.Run(ctx); err != nil {
		logger.Error("exporter stopped with error", slog.Any("error", err))
		panic(err)
	}
}
