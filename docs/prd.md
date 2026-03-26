# ndc-loader — Product Requirements Document

**Version:** 0.1.0
**Created:** 2026-03-25
**Author:** calebdunn
**Status:** Draft

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

### In Scope (MVP)

- Download and ingest the FDA NDC Directory (`ndctext.zip`) daily
- PostgreSQL storage with product and package tables
- REST API for NDC lookup, search, and package enumeration
- Full-text search across brand names, generic names, and manufacturer
- Health check endpoint with data freshness info
- Prometheus metrics endpoint
- Docker Compose deployment
- drug-cash integration as an upstream data source

### Out of Scope

- openFDA enrichment (adverse events, recalls, labels — drug-cash already caches these)
- Drug interaction data (separate data source, separate service)
- Historical NDC tracking (point-in-time snapshots)
- Write API (data comes exclusively from FDA bulk download)
- Authentication / API keys (internal network only, same as drug-cash)
- UI / dashboard

## 5. Architecture

### Tech Stack

| Layer | Technology | Version | Notes |
|-------|-----------|---------|-------|
| Language | Go | 1.22+ | Consistency with drug-cash, drug-gate |
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
| Staging | Pre-production validation | TBD | Push to staging branch |
| Production | Live service | TBD | Merge to main |

**Environment Tier:** 3 (local + staging + production)

### Data Model

See `ndc-loader-prd.md` sections 5-6 for full schema and API contract definitions.

## 6. Milestones & Roadmap

### Current Maturity: Alpha

### Roadmap

| Milestone | Goal | Target Maturity | Status | Success Criteria |
|-----------|------|-----------------|--------|------------------|
| M1: Schema + Loader | Download ZIP, parse, load into PostgreSQL, daily cron | alpha | NOW | Data loaded, row counts match FDA source |
| M2: Query API | NDC lookup, search, package enumeration endpoints | alpha | NEXT | All endpoints return correct data, <5ms lookup |
| M3: Ops | Health check, Prometheus metrics, Docker Compose, structured logging | alpha | NEXT | Health endpoint live, metrics scraped |
| M4: drug-cash Integration | Add ndc-loader as upstream in drug-cash config | alpha | LATER | Cross-slug search includes NDC data |

### Milestone Detail

#### M1: Schema + Loader [NOW]
**Goal:** Download and ingest the FDA NDC Directory into PostgreSQL
**Appetite:** 2-3 days
**Target maturity:** alpha
**Features:**
- fda-download — Download and unzip ndctext.zip from FDA
- db-schema — PostgreSQL schema with products and packages tables
- data-loader — Parse tab-delimited files and bulk load into database
- scheduler — Daily cron-triggered data refresh
**Success criteria:**
- [ ] FDA ZIP downloaded and parsed successfully
- [ ] Products and packages tables populated
- [ ] Row counts match FDA source files
- [ ] Atomic swap — consumers never see partial data
- [ ] Error handling: abort if row count drops >20%

#### M2: Query API [NEXT]
**Goal:** REST API for NDC lookup, search, and package enumeration
**Appetite:** 2-3 days
**Target maturity:** alpha
**Features:**
- ndc-lookup — GET /api/ndc/{ndc} with format normalization
- ndc-search — GET /api/ndc/search with full-text search
- package-list — GET /api/ndc/{ndc}/packages
- stats — GET /api/ndc/stats
**Success criteria:**
- [ ] Single NDC lookup < 5ms P95
- [ ] Search < 50ms P95
- [ ] NDC format normalization works (any format accepted)

#### M3: Ops [NEXT]
**Goal:** Production-readiness: health, metrics, logging, containerization
**Appetite:** 1-2 days
**Target maturity:** alpha
**Features:**
- health-check — GET /health with data freshness
- prometheus — GET /metrics with all defined metrics
- structured-logging — JSON logging with configurable level
- docker-compose — Multi-service compose file
**Success criteria:**
- [ ] Health endpoint reports data age
- [ ] Prometheus metrics scraped
- [ ] Structured JSON logs

#### M4: drug-cash Integration [LATER]
**Goal:** Add ndc-loader as upstream data source in drug-cash
**Appetite:** 1 day
**Target maturity:** alpha
**Features:**
- drug-cash-config — YAML upstream configuration for ndc-loader
**Success criteria:**
- [ ] Cross-slug search includes NDC data
- [ ] Old fda-ndc slug can be deprecated

### Maturity Promotion Path

| From | To | Requirements |
|------|-----|-------------|
| alpha -> beta | Feature specs for all endpoints, 50%+ test coverage, PR workflow, 2+ environments, TDD evidence |
| beta -> ga | 90%+ coverage, all quality gates passing, E2E tests, release tags, branch protection |

## 7. Key Features

### Feature 1: FDA Data Ingestion
Daily automated download of the FDA NDC Directory bulk ZIP, parsing of tab-delimited product.txt and package.txt files, and atomic bulk load into PostgreSQL.

### Feature 2: NDC Lookup API
REST endpoint for single NDC lookup with format normalization — accepts any common NDC format (hyphenated, unhyphenated, 2-segment, 3-segment) and returns full product details with packages.

### Feature 3: Full-Text Search
PostgreSQL ts_query-based search across brand names, generic names, and manufacturer. Supports prefix matching, relevance ranking, and pagination.

### Feature 4: drug-cash Integration
Upstream configuration allowing drug-cash to proxy ndc-loader for NDC search and lookup, integrating into existing cross-slug search.

## 8. Non-Functional Requirements

- **Performance:** Single NDC lookup < 5ms P95. Search < 50ms P95. Bulk load < 2 minutes.
- **Reliability:** Failed loads keep existing data. Never serve empty/partial dataset. Abort if row count drops >20%.
- **Observability:** Structured JSON logging. Prometheus metrics. Data freshness in health check.
- **Storage:** PostgreSQL with ~500MB disk for full dataset + indexes.
- **Security:** Internal network only. No auth required in v1. No PII in NDC data.

## 9. Open Questions

- Should the loader expose a bulk export endpoint (`GET /api/ndc/export?format=csv`) for downstream analytics?
- Include excluded NDCs (`NDC_EXCLUDE_FLAG=Y`)? ~13K products — bulk ingredients, compounding components. Exclude by default, include via query param?
- NDC format normalization depth — FDA uses multiple formats (4-4-2, 5-3-2, 5-4-1). Store canonical + accept any?

## 10. Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-03-25 | 0.1.0 | calebdunn | Initial draft from /add:init interview |
