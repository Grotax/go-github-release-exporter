package exporter

// Config contains runtime configuration for the exporter.
type Config struct {
	ListenAddress string
	MetricsPath   string
}

// DefaultConfig returns a minimal default configuration.
func DefaultConfig() Config {
	return Config{
		ListenAddress: ":9100",
		MetricsPath:   "/metrics",
	}
}
