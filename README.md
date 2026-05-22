# go-github-release-exporter

A Prometheus exporter that exposes GitHub release asset download counts for a configured repository.

## Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `github_release_asset_download_count` | Gauge | Download count per release asset |
| `github_release_published_timestamp_seconds` | Gauge | Release publish time as Unix seconds |
| `github_release_asset_download_scrape_success` | Gauge | `1` if last scrape succeeded, `0` otherwise |

Labels: `owner`, `repo`, `release_title`, `release_tag`, `artifact_name`

## Configuration

All configuration is done via environment variables.

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `GITHUB_OWNER` | — | ✅ | GitHub user or organization name |
| `GITHUB_REPO` | — | ✅ | GitHub repository name |
| `GITHUB_TOKEN` | — | — | Personal access token (avoids rate limits) |
| `GITHUB_RELEASE_LIMIT` | `0` | — | Max number of releases to fetch (`0` = unlimited) |
| `GITHUB_CACHE_TTL` | `5m` | — | How long to cache release data |
| `GITHUB_HTTP_TIMEOUT` | `15s` | — | HTTP timeout for GitHub API requests |
| `EXPORTER_LISTEN_ADDRESS` | `:9101` | — | Address the exporter listens on |
| `EXPORTER_METRICS_PATH` | `/metrics` | — | Path to expose metrics on |
| `EXPORTER_LOG_LEVEL` | `info` | — | Log level (`debug`, `info`, `warn`, `error`) |

## Usage

### Docker Compose

```yaml
services:
  github-release-exporter:
    image: ghcr.io/grotax/go-github-release-exporter:latest
    ports:
      - "9101:9101"
    environment:
      GITHUB_OWNER: "your-org"
      GITHUB_REPO: "your-repo"
      GITHUB_TOKEN: "your-token"
    restart: unless-stopped
```

### Docker

```sh
docker run -p 9101:9101 \
  -e GITHUB_OWNER=your-org \
  -e GITHUB_REPO=your-repo \
  ghcr.io/grotax/go-github-release-exporter:latest
```

### Build from source

```sh
go build -o go-github-release-exporter .
GITHUB_OWNER=your-org GITHUB_REPO=your-repo ./go-github-release-exporter
```

## Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: github-release-exporter
    static_configs:
      - targets: ["localhost:9101"]
```

## License

[MIT](LICENSE)
