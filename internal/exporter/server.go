package exporter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server owns Prometheus registry and HTTP endpoint lifecycle.
type Server struct {
	cfg      Config
	registry *prometheus.Registry
	server   *http.Server
	logger   *slog.Logger
}

// NewServer creates an exporter server with an isolated registry.
func NewServer(cfg Config, logger *slog.Logger) *Server {
	registry := prometheus.NewRegistry()
	if logger == nil {
		logger = NewLogger(slog.LevelInfo)
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.MetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return &Server{
		cfg:      cfg,
		registry: registry,
		logger:   logger,
		server: &http.Server{
			Addr:    cfg.ListenAddress,
			Handler: mux,
		},
	}
}

// RegisterCollector registers a custom collector with the server registry.
func (s *Server) RegisterCollector(collector prometheus.Collector) error {
	if collector == nil {
		return errors.New("collector is nil")
	}

	if err := s.registry.Register(collector); err != nil {
		return fmt.Errorf("register collector: %w", err)
	}

	return nil
}

// Run starts HTTP serving and shuts down gracefully when context is canceled.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting exporter", "listen_address", s.cfg.ListenAddress, "metrics_path", s.cfg.MetricsPath)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown exporter: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("run exporter: %w", err)
		}
		return nil
	}
}
