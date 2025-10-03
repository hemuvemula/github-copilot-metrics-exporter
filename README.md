# GitHub Copilot Metrics Exporter

A Prometheus exporter for GitHub Copilot metrics that scrapes the [GitHub Copilot Metrics API](https://docs.github.com/en/rest/copilot/copilot-metrics?apiVersion=2022-11-28) and exposes the data in Prometheus format.

## Features

- Exports GitHub Copilot usage metrics for organizations, teams, or enterprises
- Provides metrics on:
  - Total suggestions and acceptances
  - Lines suggested and accepted
  - Active users
  - Chat acceptances and turns
  - Active chat users
  - Acceptance rate (calculated metric)
- Easy configuration via environment variables
- Health check endpoint
- Compatible with Prometheus and Grafana

## Installation

### From Source

```bash
git clone https://github.com/hemuvemula/github-copilot-metrics-exporter.git
cd github-copilot-metrics-exporter
go build -o github-copilot-metrics-exporter .
```

### Using Go Install

```bash
go install github.com/hemuvemula/github-copilot-metrics-exporter@latest
```

## Configuration

The exporter is configured using environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_TOKEN` | Yes | GitHub Personal Access Token with appropriate permissions |
| `GITHUB_ORG` | Conditional | GitHub organization name (required if `GITHUB_ENTERPRISE` is not set) |
| `GITHUB_TEAM` | No | GitHub team slug (optional, for team-specific metrics) |
| `GITHUB_ENTERPRISE` | Conditional | GitHub enterprise name (required if `GITHUB_ORG` is not set) |
| `PORT` | No | Port to listen on (default: 8082) |
| `SCRAPE_INTERVAL` | No | Interval in seconds to fetch metrics from GitHub API (default: 3600 - 1 hour) |

### GitHub Token Permissions

Your GitHub token needs the following permissions:
- For organizations: `manage_billing:copilot` or `read:org`
- For enterprises: `manage_billing:enterprise`

## Usage

### Basic Usage (Organization)

```bash
export GITHUB_TOKEN="your_github_token"
export GITHUB_ORG="your_organization"
./github-copilot-metrics-exporter
```

### Team-Specific Metrics

```bash
export GITHUB_TOKEN="your_github_token"
export GITHUB_ORG="your_organization"
export GITHUB_TEAM="your_team_slug"
./github-copilot-metrics-exporter
```

### Enterprise Metrics

```bash
export GITHUB_TOKEN="your_github_token"
export GITHUB_ENTERPRISE="your_enterprise"
./github-copilot-metrics-exporter
```

### Custom Port and Scrape Interval

```bash
export GITHUB_TOKEN="your_github_token"
export GITHUB_ORG="your_organization"
export PORT="8080"
export SCRAPE_INTERVAL="1800"  # 30 minutes
./github-copilot-metrics-exporter
```

## How It Works

The exporter fetches GitHub Copilot metrics from the GitHub API at a configurable interval (default: 1 hour) and caches the results. When Prometheus scrapes the `/metrics` endpoint, it serves the cached data. This approach:

- Reduces API calls to GitHub (respecting rate limits)
- Provides consistent data across multiple Prometheus scrapes
- Ensures fresh data is fetched automatically at the configured interval

## Endpoints

- `/` - Landing page with links
- `/metrics` - Prometheus metrics endpoint
- `/health` - Health check endpoint

## Exported Metrics

All metrics include labels `day` (date) and `org` (organization or enterprise name).

| Metric Name | Type | Description |
|-------------|------|-------------|
| `github_copilot_suggestions_total` | Gauge | Total number of Copilot suggestions |
| `github_copilot_acceptances_total` | Gauge | Total number of Copilot acceptances |
| `github_copilot_lines_suggested_total` | Gauge | Total number of lines suggested by Copilot |
| `github_copilot_lines_accepted_total` | Gauge | Total number of lines accepted from Copilot |
| `github_copilot_active_users_total` | Gauge | Total number of active Copilot users |
| `github_copilot_chat_acceptances_total` | Gauge | Total number of Copilot chat acceptances |
| `github_copilot_chat_turns_total` | Gauge | Total number of Copilot chat turns |
| `github_copilot_active_chat_users_total` | Gauge | Total number of active Copilot chat users |
| `github_copilot_acceptance_rate` | Gauge | Copilot acceptance rate (acceptances/suggestions) |

## Example Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'github-copilot'
    scrape_interval: 5m
    static_configs:
      - targets: ['localhost:8082']
```

## Docker

### Build Docker Image

```bash
docker build -t github-copilot-metrics-exporter .
```

### Run with Docker

```bash
docker run -d \
  -p 8082:8082 \
  -e GITHUB_TOKEN="your_github_token" \
  -e GITHUB_ORG="your_organization" \
  -e SCRAPE_INTERVAL="3600" \
  github-copilot-metrics-exporter
```

## Example Queries

### Average acceptance rate over the last 7 days
```promql
avg_over_time(github_copilot_acceptance_rate{org="your_org"}[7d])
```

### Total suggestions in the last 24 hours
```promql
sum(github_copilot_suggestions_total{org="your_org"})
```

### Active users trend
```promql
github_copilot_active_users_total{org="your_org"}
```

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o github-copilot-metrics-exporter .
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
