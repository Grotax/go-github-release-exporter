package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	githubAPIBaseURL = "https://api.github.com"
	releasesPageSize = 100
)

// GitHubReleasesCollectorConfig configures the GitHub release collector.
type GitHubReleasesCollectorConfig struct {
	Owner        string
	Repo         string
	Token        string
	ReleaseLimit int
	CacheTTL     time.Duration
	HTTPTimeout  time.Duration
}

type releaseAssetDownload struct {
	ReleaseTitle  string
	ReleaseTag    string
	AssetName     string
	DownloadCount float64
}

type cacheSnapshot struct {
	fetchedAt time.Time
	entries   []releaseAssetDownload
}

type githubRelease struct {
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name          string `json:"name"`
		DownloadCount int64  `json:"download_count"`
	} `json:"assets"`
}

// GitHubReleasesCollector is a placeholder for future GitHub release metrics.
type GitHubReleasesCollector struct {
	cfg    GitHubReleasesCollectorConfig
	client *http.Client
	logger *slog.Logger
	mu     sync.Mutex
	cache  cacheSnapshot

	downloadCountDesc *prometheus.Desc
	scrapeSuccessDesc *prometheus.Desc
}

// NewGitHubReleasesCollector creates an empty collector skeleton.
func NewGitHubReleasesCollector(cfg GitHubReleasesCollectorConfig, logger *slog.Logger) *GitHubReleasesCollector {
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 5 * time.Minute
	}

	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 15 * time.Second
	}

	return &GitHubReleasesCollector{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.HTTPTimeout},
		logger: logger,
		downloadCountDesc: prometheus.NewDesc(
			"github_release_asset_download_count",
			"Download count per GitHub release asset.",
			[]string{"owner", "repo", "release_title", "release_tag", "artifact_name"},
			nil,
		),
		scrapeSuccessDesc: prometheus.NewDesc(
			"github_release_asset_download_scrape_success",
			"Whether the last scrape of GitHub releases was successful (1=success, 0=error).",
			[]string{"owner", "repo"},
			nil,
		),
	}
}

// Describe sends metric descriptors to Prometheus.
func (c *GitHubReleasesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.downloadCountDesc
	ch <- c.scrapeSuccessDesc
}

// Collect sends the current metric values to Prometheus.
func (c *GitHubReleasesCollector) Collect(ch chan<- prometheus.Metric) {
	entries, err := c.loadReleaseAssets()
	if err != nil {
		c.logger.Error("failed to collect GitHub release data", "error", err)
		ch <- prometheus.MustNewConstMetric(c.scrapeSuccessDesc, prometheus.GaugeValue, 0, c.cfg.Owner, c.cfg.Repo)
		return
	}

	for _, e := range entries {
		ch <- prometheus.MustNewConstMetric(
			c.downloadCountDesc,
			prometheus.GaugeValue,
			e.DownloadCount,
			c.cfg.Owner,
			c.cfg.Repo,
			e.ReleaseTitle,
			e.ReleaseTag,
			e.AssetName,
		)
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeSuccessDesc, prometheus.GaugeValue, 1, c.cfg.Owner, c.cfg.Repo)
}

func (c *GitHubReleasesCollector) loadReleaseAssets() ([]releaseAssetDownload, error) {
	c.mu.Lock()
	if !c.cache.fetchedAt.IsZero() && time.Since(c.cache.fetchedAt) < c.cfg.CacheTTL {
		cached := make([]releaseAssetDownload, len(c.cache.entries))
		copy(cached, c.cache.entries)
		c.mu.Unlock()
		c.logger.Debug("using cached GitHub release data", "age_seconds", int(time.Since(c.cache.fetchedAt).Seconds()))
		return cached, nil
	}
	c.mu.Unlock()

	c.logger.Info("refreshing GitHub release data", "owner", c.cfg.Owner, "repo", c.cfg.Repo, "release_limit", c.cfg.ReleaseLimit)

	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.HTTPTimeout)
	defer cancel()

	releases, err := c.fetchReleases(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]releaseAssetDownload, 0)
	for _, release := range releases {
		releaseTitle := strings.TrimSpace(release.Name)
		releaseTag := strings.TrimSpace(release.TagName)
		if releaseTitle == "" {
			releaseTitle = releaseTag
		}
		for _, asset := range release.Assets {
			entries = append(entries, releaseAssetDownload{
				ReleaseTitle:  releaseTitle,
				ReleaseTag:    releaseTag,
				AssetName:     asset.Name,
				DownloadCount: float64(asset.DownloadCount),
			})
		}
	}

	c.mu.Lock()
	c.cache = cacheSnapshot{
		fetchedAt: time.Now(),
		entries:   entries,
	}
	c.mu.Unlock()

	c.logger.Info("refreshed GitHub release data", "release_count", len(releases), "asset_count", len(entries))
	return entries, nil
}

func (c *GitHubReleasesCollector) fetchReleases(ctx context.Context) ([]githubRelease, error) {
	releases := make([]githubRelease, 0)
	page := 1

	for {
		if c.cfg.ReleaseLimit > 0 && len(releases) >= c.cfg.ReleaseLimit {
			return releases[:c.cfg.ReleaseLimit], nil
		}

		url := fmt.Sprintf(
			"%s/repos/%s/%s/releases?per_page=%d&page=%d",
			githubAPIBaseURL,
			c.cfg.Owner,
			c.cfg.Repo,
			releasesPageSize,
			page,
		)

		batch, err := c.fetchReleasePage(ctx, url)
		if err != nil {
			return nil, err
		}

		if len(batch) == 0 {
			break
		}

		releases = append(releases, batch...)
		c.logger.Debug("fetched GitHub releases page", "page", page, "count", len(batch))

		if len(batch) < releasesPageSize {
			break
		}

		page++
	}

	if c.cfg.ReleaseLimit > 0 && len(releases) > c.cfg.ReleaseLimit {
		return releases[:c.cfg.ReleaseLimit], nil
	}

	return releases, nil
}

func (c *GitHubReleasesCollector) fetchReleasePage(ctx context.Context, url string) ([]githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "go-github-release-exporter")
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		rateRemaining := resp.Header.Get("X-RateLimit-Remaining")
		rateReset := resp.Header.Get("X-RateLimit-Reset")
		c.logger.Warn(
			"GitHub API non-200 response",
			"status_code", resp.StatusCode,
			"rate_limit_remaining", rateRemaining,
			"rate_limit_reset_unix", rateReset,
		)
		return nil, fmt.Errorf("github API status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if remainingInt, convErr := strconv.Atoi(remaining); convErr == nil && remainingInt < 10 {
			c.logger.Warn("GitHub API rate limit running low", "remaining", remainingInt)
		}
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode releases response: %w", err)
	}

	return releases, nil
}
