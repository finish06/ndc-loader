# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Added
- Health endpoint with postgres dependency check, uptime, and data freshness (`/health`)
- Version endpoint with build metadata injected via ldflags (`/version`)
- Structured pharmacological classification (`pharm_classes_structured`) on internal API responses
- Optional pgAdmin UI service in staging compose (`--profile debug`)
- Swagger UI with auto-generated OpenAPI spec at `/swagger/`
- openFDA-compatible endpoint (`/api/openfda/ndc.json`) — drop-in replacement for drug-cash
- NDC lookup with full format normalization (4-4-2, 5-3-2, 5-4-1 patterns)
- Full-text search with prefix matching and relevance ranking
- Package enumeration endpoint (`/api/ndc/{ndc}/packages`)
- Dataset statistics endpoint (`/api/ndc/stats`)
- Admin endpoints for manual data load trigger and status tracking
- API key authentication via `X-API-Key` header
- FDA data pipeline: download, extract, parse, bulk load with atomic swap
- Checkpoint-based retry — resume from last successful table on failure
- Row count safety valve — abort if data drops >20%
- UTF-8 sanitization for Windows-1252 bytes in FDA data
- Prometheus metrics endpoint (`/metrics`)
- Cron-based daily FDA data refresh (configurable schedule)
- GitHub Actions CI pipeline (lint, test, build, publish, staging deploy)
- Staging deployment to 192.168.1.145 via webhook
- Docker Compose for local dev and staging
- Makefile with build, test, lint, deploy, and staging operations targets

### Fixed
- CI publish job needs `contents:read` permission for private repo checkout
- Generated swagger docs must be committed (main.go blank import references the package)
- `.gitignore` pattern `ndc-loader` excluded `cmd/ndc-loader/` directory — changed to `/ndc-loader`
- Go version in Dockerfile updated from 1.22 to 1.26 to match local toolchain
- `TrimLeadingSpace` in Go csv.Reader collapsed empty FDA tab-delimited fields
- pgx binary encoding requires native Go types (time.Time, bool) not strings for DATE/BOOL columns
- FDA data contains orphaned FK references between datasets — removed hard FK constraints
- Real FDA file structure differs from spec (PROPRIETARYNAMESUFFIX, SponsorName, SubmissionClassCodeID)

### Changed
- Streaming parser for single-pass file processing (reduces peak memory)
- O(1) atomic table swap via DROP + RENAME instead of INSERT SELECT
- Batch package loading eliminates N+1 queries in openFDA search
- Docker image name is `rx-dag` (not `ndc-loader`)
- Staging deploy directory is `/opt/rx-dag`
- QueryProvider and CheckpointStoreProvider interfaces for testability

### Documentation
- Comprehensive README with Mermaid flow charts (system, pipeline, request flow, data model, checkpoints)
- API reference (`docs/api/endpoints.md`) with examples for all endpoints
- Deployment guide (`docs/deployment.md`) — local, staging, CI/CD, release process
- Data model documentation (`docs/data-model.md`) — tables, joins, pharm classification
- Feature specs for all implemented features
