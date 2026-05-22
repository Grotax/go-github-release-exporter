package collectors

import "github.com/prometheus/client_golang/prometheus"

// GitHubReleasesCollector is a placeholder for future GitHub release metrics.
type GitHubReleasesCollector struct{}

// NewGitHubReleasesCollector creates an empty collector skeleton.
func NewGitHubReleasesCollector() *GitHubReleasesCollector {
	return &GitHubReleasesCollector{}
}

// Describe sends metric descriptors to Prometheus.
// TODO: Add descriptor definitions when collector logic is implemented.
func (c *GitHubReleasesCollector) Describe(ch chan<- *prometheus.Desc) {
	// Intentionally empty for skeleton stage.
}

// Collect sends the current metric values to Prometheus.
// TODO: Add collection logic when exporter behavior is defined.
func (c *GitHubReleasesCollector) Collect(ch chan<- prometheus.Metric) {
	// Intentionally empty for skeleton stage.
}
