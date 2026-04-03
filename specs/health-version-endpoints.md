# Spec: Health & Version Endpoints

**Version:** 0.1.0
**Created:** 2026-03-29
**PRD Reference:** docs/prd.md
**Status:** Done
**Completed:** 2026-03-29
**Milestone:** M1 — Schema & Loader

## 1. Overview

Two operational endpoints that expose service health and build information. Both are unauthenticated (no API key required) for use by load balancers, monitoring, and deploy hooks.

- **GET /version** — Build and runtime metadata (git commit, branch, Go version, OS, arch, deployment tag)
- **GET /health** — Service health with dependency checks, uptime, and version summary

### User Story

As a **platform operator or deploy hook**, I want to query `/health` and `/version` to verify the service is running correctly and which version is deployed, so that I can detect failed deploys and stale instances.

## 2. Acceptance Criteria

| ID | Criterion | Priority |
|----|-----------|----------|
| AC-001 | GET /version returns git_branch, git_commit, go_version, version tag, os, architecture | Must |
| AC-002 | Build info is injected at compile time via -ldflags (not hardcoded) | Must |
| AC-003 | GET /health returns status (ok/degraded/error), version tag, uptime, start_time | Must |
| AC-004 | GET /health includes dependency checks array with postgres status | Must |
| AC-005 | Postgres dependency check pings the DB and reports connected/disconnected with latency | Must |
| AC-006 | GET /health status is "degraded" when data is >48h stale | Must |
| AC-007 | GET /health status is "error" when postgres is disconnected | Must |
| AC-008 | Both endpoints require no authentication | Must |
| AC-009 | GET /health includes data_age_hours and last_load timestamp | Should |
| AC-010 | Response times for both endpoints are <10ms | Should |

## 3. API Contract

### GET /version

```json
{
  "version": "0.1.0",
  "git_commit": "7685dcc",
  "git_branch": "main",
  "go_version": "go1.26.1",
  "os": "linux",
  "arch": "amd64",
  "build_time": "2026-03-28T14:55:00Z"
}
```

### GET /health

```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime": "4h32m15s",
  "start_time": "2026-03-29T03:00:00Z",
  "data_age_hours": 4.2,
  "last_load": "2026-03-29T03:15:00Z",
  "dependencies": [
    {
      "name": "postgres",
      "status": "connected",
      "latency_ms": 1.2
    }
  ]
}
```

**Status logic:**
- `"ok"` — postgres connected, data fresh (<48h)
- `"degraded"` — postgres connected, data stale (>48h) or never loaded
- `"error"` — postgres disconnected

### GET /health (error state)

```json
{
  "status": "error",
  "version": "0.1.0",
  "uptime": "0h2m30s",
  "start_time": "2026-03-29T03:00:00Z",
  "data_age_hours": null,
  "last_load": null,
  "dependencies": [
    {
      "name": "postgres",
      "status": "disconnected",
      "latency_ms": null,
      "error": "connection refused"
    }
  ]
}
```

## 4. Build-Time Injection

Version info injected via `go build -ldflags`:

```bash
go build -ldflags="\
  -X main.version=0.1.0 \
  -X main.gitCommit=$(git rev-parse --short HEAD) \
  -X main.gitBranch=$(git rev-parse --abbrev-ref HEAD) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -s -w" \
  -o ndc-loader ./cmd/ndc-loader
```

Dockerfile and CI updated to pass these at build time.

## 5. Dependencies

- `runtime` package (GOOS, GOARCH, Go version)
- `pgxpool.Ping()` for postgres health check
- `time.Since(startTime)` for uptime

## 6. Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-03-29 | 0.1.0 | calebdunn | Initial spec |
