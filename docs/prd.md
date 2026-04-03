# ndc-loader — Product Requirements Document

**Version:** 0.2.0
**Created:** 2026-03-25
**Updated:** 2026-04-01
**Author:** calebdunn
**Status:** Active

## 1. Problem Statement

Internal services need to look up drug information by NDC (National Drug Code) — product details, package configurations, manufacturer info, pharmacological classes. Today, drug-cash proxies the openFDA `/drug/ndc.json` API, but this is rate-limited, incomplete, doesn't support joins, and has no offline capability.

The FDA publishes the complete NDC Directory as a daily bulk download (~140K products, ~250K packages). By ingesting this locally, we get the full dataset with zero external API dependency at query time.

## 2. Target Users

- **drug-cash:** Proxies ndc-loader as an upstream, giving consumers a single entry point for all drug data
- **Internal microservices:** Any service that needs NDC lookup, drug name search, or package enumeration
- **Future consumers:** Analytics, reporting, compliance tools that need bulk drug data access

## 3. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| NDC lookup latency | < 5ms P95 | Query response time for single NDC lookup |
| Search latency | < 50ms P95 | Full-text search across drug names |
| Data freshness | < 24 hours | Time since last successful FDA data load |
| Coverage | 100% of FDA NDC directory | Row count matches FDA source file |
| Availability | 99.9% | Uptime excluding planned maintenance |

## 4. Scope

### In Scope (MVP) — All Delivered

- Download and ingest the FDA NDC Directory (`ndctext.zip`) and Drugs@FDA datasets daily
- PostgreSQL storage with 7 tables (products, packages, applications, drugsfda_products, submissions, marketing_status, te_codes)
- REST API for NDC lookup, search, and package enumeration
- Full-text search across brand names, generic names, and manufacturer
- openFDA-compatible endpoint (`/api/openfda/ndc.json`) — drop-in replacement for drug-cash
- API key authentication via `X-API-Key` header
- Health check endpoint with postgres dependency check, uptime, and data freshness
- Version endpoint with build metadata
- Prometheus metrics endpoint
- Swagger UI with auto-generated OpenAPI spec at `/swagger/`
- Docker Compose deployment (local + staging)
- GitHub Actions CI/CD with staging webhook deploy

### Out of Scope

- openFDA enrichment (adverse events, recalls, labels — drug-cash already caches these)
- Drug interaction data (separate data source, separate service)
- Historical NDC tracking (point-in-time snapshots)
- Write API (data comes exclusively from FDA bulk download)
- UI / dashboard (beyond Swagger UI for API docs)

## 5. Architecture

### Tech Stack

| Layer | Technology | Version | Notes |
|-------|-----------|---------|-------|
| Language | Go | 1.26+ | Consistency with drug-cash, drug-gate |
| HTTP Framework | Chi | v5 | Lightweight router with middleware support |
| Database | PostgreSQL | 16+ | Full-text search via GIN indexes |
| DB Driver | pgx | v5 | Native Go PostgreSQL driver |
| Containers | Docker Compose | — | Match drug-cash deployment pattern |
| Metrics | Prometheus client | — | `/metrics` endpoint |

### Infrastructure

| Component | Choice | Notes |
|-----------|--------|-------|
| Git Host | GitHub | Remote TBD |
| Cloud Provider | Self-hosted | Homelab, dockerhub.calebdunn.tech |
| CI/CD | GitHub Actions | .github/workflows/ci.yml |
| Containers | Docker Compose | Local dev + production deploy |
| IaC | None | Docker Compose is the deployment unit |

### Environment Strategy

| Environment | Purpose | URL | Deploy Trigger |
|-------------|---------|-----|----------------|
| Local | Development & unit tests | http://localhost:8081 | Manual (docker-compose up) |
| Staging | Pre-production validation | http://192.168.1.145:8081 | Push to main (auto via webhook) |
| Production | Live service | TBD | Tag `v*` (manual) |

**Environment Tier:** 3 (local + staging + production)

### Data Model

See `ndc-loader-prd.md` sections 5-6 for full schema and API contract definitions.

## 6. Milestones & Roadmap

### Current Maturity: Alpha

### Roadmap

| Milestone | Goal | Target Maturity | Status | Success Criteria |
|-----------|------|-----------------|--------|------------------|
| M1: Schema + Loader | Download, parse, load FDA data; query API with search | alpha | COMPLETE | 112K products, 212K packages, 29K applications loaded; all query endpoints verified |
| M2: drug-cash Integration | openFDA-compatible API as drop-in replacement | alpha | COMPLETE | 14/14 AC, format parity verified against live openFDA |
| M3: Swagger Docs | Auto-generated OpenAPI spec with Swagger UI | alpha | COMPLETE | Swagger UI at /swagger/, all endpoints documented |
| M4: Production Readiness | Release tagging, production deploy, monitoring, public landing page | beta | NEXT | Tagged release, production deployment, SLA monitoring, rx-dag landing page live on GitHub Pages |

### Milestone Detail

#### M1: Schema + Loader [COMPLETE]
**Goal:** Download and ingest the FDA NDC Directory and Drugs@FDA datasets into PostgreSQL with checkpoint-based retry and API key authentication
**Completed:** 2026-03-26
**Features:**
- fda-data-fetcher — Download, extract, parse, bulk load with atomic swap, checkpoint retry
- query-api — NDC lookup, full-text search, package enumeration, stats endpoint
**Results:**
- [x] 112K products, 212K packages, 29K applications loaded
- [x] Atomic swap — consumers never see partial data
- [x] Row count safety valve (abort if >20% drop)
- [x] NDC format normalization (4-4-2, 5-3-2, 5-4-1 patterns)
- [x] Full-text search with prefix matching and relevance ranking

#### M2: drug-cash Integration [COMPLETE]
**Goal:** Make ndc-loader a drop-in upstream replacement for the openFDA NDC API in drug-cash
**Completed:** 2026-03-27
**Features:**
- openfda-compat-api — `/api/openfda/ndc.json` with openFDA search syntax and response format
**Results:**
- [x] 14/14 acceptance criteria met
- [x] Format parity verified against live openFDA API
- [x] Supports: field search, exact phrases, AND queries, pagination

#### M3: Swagger Docs [COMPLETE]
**Goal:** Auto-generated OpenAPI 3.0 docs served via Swagger UI
**Completed:** 2026-04-01
**Features:**
- swagger-docs — swaggo/swag annotations, Swagger UI at /swagger/, all endpoints documented
**Results:**
- [x] Swagger UI loads at /swagger/
- [x] All endpoints documented with params, responses, examples
- [x] API key auth scheme documented
- [x] No authentication required for Swagger UI access

#### M4: Production Readiness [NEXT]
**Goal:** Stable release with production deployment, monitoring, and public landing page
**Appetite:** 1 week
**Target maturity:** beta
**Features:**
- Release tagging with semantic versioning
- Production deployment workflow
- SLA monitoring and alerting
- Performance baselines
- rx-dag landing page on GitHub Pages with config-driven root redirect
**Success criteria:**
- [ ] Tagged v1.0.0 release
- [ ] Production deploy via tagged release
- [ ] Response time baselines established
- [ ] Uptime monitoring configured
- [ ] rx-dag landing page live on GitHub Pages
- [ ] GET / redirects to landing page via LANDING_URL env var

### Maturity Promotion Path

| From | To | Requirements |
|------|-----|-------------|
| alpha -> beta | Feature specs for all endpoints, 50%+ test coverage, PR workflow, 2+ environments, TDD evidence |
| beta -> ga | 90%+ coverage, all quality gates passing, E2E tests, release tags, branch protection |

## 7. Key Features

### Feature 1: FDA Data Ingestion
Daily automated download of the FDA NDC Directory and Drugs@FDA datasets, parsing of tab-delimited files with UTF-8 sanitization, and atomic bulk load into PostgreSQL with checkpoint-based retry and row count safety valve.

### Feature 2: NDC Lookup API
REST endpoint for single NDC lookup with format normalization — accepts any common NDC format (hyphenated, unhyphenated, 2-segment, 3-segment) and returns full product details with packages and structured pharmacological classification.

### Feature 3: Full-Text Search
PostgreSQL ts_query-based search across brand names, generic names, and manufacturer. Supports prefix matching, relevance ranking, and pagination.

### Feature 4: openFDA-Compatible API
Drop-in replacement for the openFDA `/drug/ndc.json` endpoint with identical response format. Supports openFDA search syntax (field:value, exact phrases, AND via `+`). Enables drug-cash to swap upstream without code changes.

### Feature 5: Interactive API Documentation
Auto-generated OpenAPI spec via swaggo/swag annotations, served through Swagger UI at `/swagger/`. All endpoints documented with parameters, response schemas, and authentication requirements.

## 8. Non-Functional Requirements

- **Performance:** Single NDC lookup < 5ms P95. Search < 50ms P95. Bulk load < 2 minutes.
- **Reliability:** Failed loads keep existing data. Never serve empty/partial dataset. Abort if row count drops >20%.
- **Observability:** Structured JSON logging. Prometheus metrics. Data freshness in health check.
- **Storage:** PostgreSQL with ~500MB disk for full dataset + indexes.
- **Security:** Internal network only. API key authentication via `X-API-Key` header. No PII in NDC data.

## 9. Open Questions

- ~~Should the loader expose a bulk export endpoint (`GET /api/ndc/export?format=csv`) for downstream analytics?~~ **Resolved:** No. Downstream consumers query the REST API directly; no bulk export endpoint will be provided.
- Include excluded NDCs (`NDC_EXCLUDE_FLAG=Y`)? ~13K products — bulk ingredients, compounding components. Exclude by default, include via query param?
- ~~NDC format normalization depth — FDA uses multiple formats (4-4-2, 5-3-2, 5-4-1). Store canonical + accept any?~~ **Resolved:** All formats accepted and normalized. Tries 4-4-2, 5-3-2, 5-4-1 patterns for unhyphenated input.

## 10. Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-03-25 | 0.1.0 | calebdunn | Initial draft from /add:init interview |
| 2026-04-01 | 0.2.0 | calebdunn | Updated to reflect M1-M3 completion, actual architecture, resolved questions |
