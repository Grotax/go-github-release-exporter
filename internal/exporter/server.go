package exporter

import (
	"context"
	"errors"
	"fmt"
	"log"
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
}

// NewServer creates an exporter server with an isolated registry.
func NewServer(cfg Config) *Server {
	registry := prometheus.NewRegistry()

	mux := http.NewServeMux()
	mux.Handle(cfg.MetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return &Server{
		cfg:      cfg,
		registry: registry,
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
		log.Printf("starting exporter on %s (metrics at %s)", s.cfg.ListenAddress, s.cfg.MetricsPath)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
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
