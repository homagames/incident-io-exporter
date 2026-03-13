# incident-io-exporter

[![CI](https://github.com/homagames/incident-io-exporter/actions/workflows/ci.yml/badge.svg)](https://github.com/homagames/incident-io-exporter/actions/workflows/ci.yml)
[![Release](https://github.com/homagames/incident-io-exporter/actions/workflows/release.yml/badge.svg)](https://github.com/homagames/incident-io-exporter/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Prometheus exporter for [incident.io](https://incident.io) — collects incidents, alerts, and severities via the incident.io API and exposes them as Prometheus metrics. Built for Grafana dashboards and alerting.

## Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `incidentio_incidents_total` | Gauge | `status`, `severity`, `type`, `mode`, `visibility`, `team` | Count of incidents |
| `incidentio_incidents_duration_seconds` | Gauge | `severity`, `metric` | Average duration metric values by severity |
| `incidentio_alerts_total` | Gauge | `status`, `team` | Count of alerts |
| `incidentio_severities` | Gauge | `name`, `rank` | Configured severities |
| `incidentio_build_info` | Gauge | `version` | Build information |
| `incidentio_scrape_duration_seconds` | Gauge | — | Time taken for the last scrape |
| `incidentio_scrape_errors_total` | Gauge | — | Total scrape errors |

## Installation

### Helm (recommended)

```bash
helm install incident-io-exporter \
  oci://ghcr.io/homagames/incident-io-exporter/charts/incident-io-exporter \
  --set existingSecret=my-incident-io-secret
```

The chart expects a Kubernetes Secret with your API key. Create it first:

```bash
kubectl create secret generic my-incident-io-secret \
  --from-literal=api-key=YOUR_API_KEY
```

#### Helm values

```yaml
replicaCount: 1

image:
  repository: ghcr.io/homagames/incident-io-exporter
  tag: ""  # Defaults to chart appVersion

service:
  type: ClusterIP
  port: 9090

env:
  LISTEN_ADDRESS: ":9090"
  METRICS_PATH: "/metrics"
  LOG_LEVEL: "info"
  SCRAPE_INTERVAL: "1m"

existingSecret: "my-incident-io-secret"
secretKey: "api-key"

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"

nodeSelector: {}
tolerations: []
affinity: {}
```

### Docker

```bash
docker run -d \
  -p 9090:9090 \
  -e INCIDENT_IO_API_KEY=your-api-key \
  ghcr.io/homagames/incident-io-exporter:latest
```

Multi-arch images (amd64/arm64) are published to `ghcr.io/homagames/incident-io-exporter`.

### Binary

Download pre-built binaries for Linux and macOS (amd64/arm64) from the [releases page](https://github.com/homagames/incident-io-exporter/releases).

```bash
INCIDENT_IO_API_KEY=your-api-key ./incident-io-exporter
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `INCIDENT_IO_API_KEY` | *(required)* | incident.io API key |
| `LISTEN_ADDRESS` | `:9090` | HTTP server listen address |
| `METRICS_PATH` | `/metrics` | Prometheus metrics endpoint path |
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `SCRAPE_INTERVAL` | `1m` | How often to query the incident.io API (min `10s`) |

### Endpoints

| Path | Description |
|------|-------------|
| `/metrics` | Prometheus metrics (configurable) |
| `/healthz` | Health check (returns `200 OK`) |
| `/` | Landing page with link to metrics |

## Network Traffic

If you run in a restricted network environment (e.g. Cilium `NetworkPolicy`, Calico, or Kubernetes `NetworkPolicy`), here is what this exporter needs:

**Ingress:**

| Source | Port | Protocol | Description |
|--------|------|----------|-------------|
| Prometheus scraper | `9090/tcp` (or your `LISTEN_ADDRESS`) | HTTP | Metrics scraping |

**Egress:**

| Destination | Port | Protocol | Description |
|-------------|------|----------|-------------|
| `api.incident.io` | `443/tcp` | HTTPS | incident.io API calls |

## Grafana Dashboard

A pre-built Grafana dashboard is included in [`dashboards/incident-io-exporter.json`](dashboards/incident-io-exporter.json). Import it via the Grafana UI or provision it with your Grafana configuration.

## incident.io API

The exporter uses the following incident.io API endpoints:

- `GET /v2/incidents` — list incidents (cursor pagination, max 250 per page)
- `GET /v2/alerts` — list alerts (cursor pagination, max 50 per page)
- `GET /v1/severities` — list configured severities

Authentication is via `Authorization: Bearer <API_KEY>`. The API has a rate limit of 1200 requests/min per key; the client retries `429` responses with exponential backoff using `Retry-After` headers.

## Contributing

### Prerequisites

- Go 1.23+
- [golangci-lint](https://golangci-lint.run/)
- [pre-commit](https://pre-commit.com/) (optional but recommended)
- [Helm](https://helm.sh/) (for chart development)

### Setup

```bash
git clone https://github.com/homagames/incident-io-exporter.git
cd incident-io-exporter
pre-commit install --hook-type pre-commit --hook-type commit-msg
```

### Build & Test

```bash
make all            # lint + test + build
make build          # build binary to ./bin/incident-io-exporter
make test           # run tests with race detector
make test-coverage  # run tests with coverage (must be ≥80%)
make lint           # run golangci-lint
make docker-build   # build container image locally
make helm-lint      # lint Helm chart
```

Run a single test:

```bash
go test -run TestCollectorDescribe ./internal/collector/
```

### Commit Convention

This project uses [Conventional Commits](https://www.conventionalcommits.org/), enforced by a pre-commit hook. Commit messages determine version bumps via [semantic-release](https://semantic-release.gitbook.io/):

| Prefix | Version bump |
|--------|-------------|
| `fix:` | Patch |
| `feat:` | Minor |
| `BREAKING CHANGE:` / `!:` | Major |

Allowed prefixes: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.

### Release Process

Releases are fully automated on the `main` branch:

1. **semantic-release** determines the next version from commit messages
2. **goreleaser** builds multi-arch binaries and Docker images
3. Container images are pushed to `ghcr.io/homagames/incident-io-exporter`
4. Helm chart is packaged and pushed to `oci://ghcr.io/homagames/incident-io-exporter/charts`

## License

MIT
