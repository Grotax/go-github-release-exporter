package exporter

import (
	"log/slog"
	"os"
)

// NewLogger returns a console logger honoring the configured log level.
func NewLogger(level slog.Level) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
