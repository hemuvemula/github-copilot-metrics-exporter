# GitHub Copilot Metrics Exporter

A Prometheus exporter for GitHub Copilot metrics that scrapes the [GitHub Copilot Metrics API](https://docs.github.com/en/rest/copilot/copilot-metrics?apiVersion=2022-11-28) and exposes the data in Prometheus format.

## Features

- Exports comprehensive GitHub Copilot usage metrics for organizations, teams, or enterprises
- **Complete API coverage**: Captures ALL data points from the GitHub Copilot Metrics API including:
  - Total suggestions, acceptances, lines suggested/accepted
  - Active users and chat metrics
  - Breakdown by language, editor, and model
  - IDE Code Completions metrics with detailed breakdowns
  - IDE Chat metrics with editor and model breakdowns
  - Dotcom Chat metrics with model breakdowns
  - Dotcom Pull Requests metrics with repository-level details
  - Acceptance rate (calculated metric)
- **Real-time data**: Fetches fresh data from GitHub API on each Prometheus scrape (no caching)
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

### Custom Port

```bash
export GITHUB_TOKEN="your_github_token"
export GITHUB_ORG="your_organization"
export PORT="8080"
./github-copilot-metrics-exporter
```

## How It Works

The exporter fetches GitHub Copilot metrics from the GitHub API on every Prometheus scrape request. This ensures you always get the most up-to-date data. The exporter captures ALL fields from the API response including:

- Aggregate daily metrics
- Breakdown by programming language, editor, and AI model
- IDE Code Completions with detailed categorization
- IDE Chat metrics
- Dotcom Chat metrics
- Dotcom Pull Requests metrics with repository-level details

## Endpoints

- `/` - Landing page with links
- `/metrics` - Prometheus metrics endpoint
- `/health` - Health check endpoint

## Exported Metrics

### Top-Level Aggregate Metrics

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

### Breakdown Metrics

These metrics include additional labels: `language`, `editor`, and `model` to provide detailed breakdowns.

| Metric Name | Type | Description |
|-------------|------|-------------|
| `github_copilot_breakdown_suggestions_total` | Gauge | Suggestions by language/editor/model |
| `github_copilot_breakdown_acceptances_total` | Gauge | Acceptances by language/editor/model |
| `github_copilot_breakdown_lines_suggested_total` | Gauge | Lines suggested by language/editor/model |
| `github_copilot_breakdown_lines_accepted_total` | Gauge | Lines accepted by language/editor/model |
| `github_copilot_breakdown_active_users` | Gauge | Active users by language/editor/model |
| `github_copilot_breakdown_chat_acceptances_total` | Gauge | Chat acceptances by language/editor/model |
| `github_copilot_breakdown_chat_turns_total` | Gauge | Chat turns by language/editor/model |
| `github_copilot_breakdown_active_chat_users` | Gauge | Active chat users by language/editor/model |

### Feature-Specific Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `github_copilot_ide_code_completions_engaged_users` | Gauge | Total engaged users for IDE code completions |
| `github_copilot_ide_chat_engaged_users` | Gauge | Total engaged users for IDE chat |
| `github_copilot_dotcom_chat_engaged_users` | Gauge | Total engaged users for Dotcom chat |
| `github_copilot_dotcom_pr_engaged_users` | Gauge | Total engaged users for Dotcom pull requests |
| `github_copilot_dotcom_pr_repo_engaged_users` | Gauge | Engaged users for Dotcom pull requests by repository (includes `repository` label) |

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

### Breakdown by programming language
```promql
github_copilot_breakdown_suggestions_total{language!=""}
```

### IDE Chat engaged users by editor
```promql
github_copilot_breakdown_active_chat_users{editor!=""}
```

### Model usage across all features
```promql
github_copilot_breakdown_suggestions_total{model!=""}
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
