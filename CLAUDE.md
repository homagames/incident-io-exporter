# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Prometheus exporter for [incident.io](https://incident.io) — collects incidents, alerts, severities, and statuses via the incident.io API and exposes them as Prometheus metrics. Designed for Grafana dashboards.

- **Language:** Go
- **Module:** `github.com/homagames/incident-io-exporter`
- **License:** MIT

## Build & Development Commands

```bash
make build          # Build binary to ./bin/incident-io-exporter
make test           # Run tests with race detector
make test-coverage  # Run tests with coverage (must be ≥80%)
make lint           # Run golangci-lint
make docker-build   # Build container image locally
make helm-lint      # Lint Helm chart
make all            # lint + test + build
```

Run a single test:
```bash
go test -run TestCollectorDescribe ./internal/collector/
```

## Architecture

```
main.go                          → CLI entry point (flags, config, starts HTTP server)
internal/
  config/config.go               → Configuration from env vars / flags
  client/                        → incident.io API client (HTTP, pagination)
    client.go                    → Base HTTP client with auth, rate limiting, pagination
    incidents.go                 → GET /v2/incidents (list with cursor pagination)
    alerts.go                    → GET /v2/alerts (list with cursor pagination)
    severities.go                → GET /v1/severities
    incident_statuses.go         → GET /v1/incident-statuses
  collector/                     → Prometheus collectors
    collector.go                 → Main collector implementing prometheus.Collector
    incidents.go                 → Incident metrics (gauges by status/severity/type)
    alerts.go                    → Alert metrics (gauges by status/source)
```

## Key Design Decisions

- **Single binary, single collector** — one `prometheus.Collector` registered with a default registry, scraped at `/metrics`.
- **API pagination** — incident.io uses cursor-based pagination (`after` field in `pagination_meta`). All list endpoints page through until exhausted.
- **Rate limiting** — incident.io allows 1200 req/min per API key. The client respects `429` responses with `Retry-After`.
- **Labels** — Prometheus metrics use low-cardinality labels only: `status`, `severity`, `type`, `mode`, `visibility`. No incident names/IDs as labels.

## Prometheus Metrics Exposed

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `incidentio_incidents_total` | Gauge | status, severity, type, mode, visibility | Count of incidents |
| `incidentio_incidents_duration_seconds` | Gauge | name, severity | Duration metric values |
| `incidentio_alerts_total` | Gauge | status | Count of alerts |
| `incidentio_severities` | Gauge | name, rank | Configured severities |
| `incidentio_scrape_duration_seconds` | Gauge | — | Time taken for last scrape |
| `incidentio_scrape_errors_total` | Counter | — | Total scrape errors |

## incident.io API Reference

- **Base URL:** `https://api.incident.io/v2/` (v1 for severities/statuses)
- **Auth:** `Authorization: Bearer <API_KEY>`
- **Rate limit:** 1200 req/min, 429 with `Retry-After` on throttle
- **Pagination:** Cursor-based via `pagination_meta.after`, max page size 250 (incidents) / 50 (alerts)

## Release & Versioning

- **Conventional Commits** enforced via pre-commit hook. Commit messages determine version bumps:
  - `fix:` → patch, `feat:` → minor, `BREAKING CHANGE:` / `!:` → major
- **semantic-release** runs on `main` branch via GitHub Actions to tag, build, and publish
- **goreleaser** builds multi-arch binaries (linux/darwin, amd64/arm64) and Docker images
- **Container:** Published to `ghcr.io/homagames/incident-io-exporter` (scratch-based)
- **Helm chart:** Published to GitHub Pages OCI registry from `charts/`
- **Grafana dashboard:** Versioned in `dashboards/` and bundled in releases

## CI/CD (GitHub Actions)

- **ci.yml** — On push/PR: lint, test (80% coverage gate), build, helm lint
- **release.yml** — On push to main: semantic-release → goreleaser → container push → helm publish
- **renovate** — Daily dependency update PRs with automerge on green CI

## Configuration (env vars)

| Variable | Default | Description |
|----------|---------|-------------|
| `INCIDENT_IO_API_KEY` | (required) | incident.io API key |
| `LISTEN_ADDRESS` | `:9090` | HTTP server listen address |
| `METRICS_PATH` | `/metrics` | Prometheus metrics endpoint path |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
