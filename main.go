package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/grotax/go-github-release-exporter/internal/collectors"
	"github.com/grotax/go-github-release-exporter/internal/exporter"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := exporter.DefaultConfig()

	srv := exporter.NewServer(cfg)

	// Register placeholder collectors here as the project grows.
	srv.RegisterCollector(collectors.NewGitHubReleasesCollector())

	if err := srv.Run(ctx); err != nil {
		log.Fatalf("exporter stopped with error: %v", err)
	}
}
