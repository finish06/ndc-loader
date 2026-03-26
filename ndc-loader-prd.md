# ndc-loader — Product Requirements Document

**Version:** 0.1.0
**Date:** 2026-03-25
**Author:** calebdunn
**Status:** Draft — for team review

## 1. Problem Statement

Internal services need to look up drug information by NDC (National Drug Code) — product details, package configurations, manufacturer info, pharmacological classes. Today, drug-cash proxies the openFDA `/drug/ndc.json` API, but this has significant limitations:

- **Rate-limited:** openFDA enforces API rate limits; bulk lookups are slow and unreliable
- **Incomplete:** API search is best-effort text matching, not exact NDC resolution
- **No joins:** Product-to-package relationships are flattened in the API response; no way to query "all packages for this product"
- **Stale on failure:** When openFDA is down, drug-cash serves cached data but can't serve NDCs it hasn't seen before
- **No offline capability:** Every new NDC lookup depends on an external API call

The FDA publishes the **complete NDC Directory** as a daily bulk download (~140K products, ~250K packages). By ingesting this locally, we get the full dataset with zero external API dependency at query time.

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

### In Scope

- Download and ingest the FDA NDC Directory (`ndctext.zip`) daily
- PostgreSQL storage with product and package tables
- REST API for NDC lookup, search, and package enumeration
- Full-text search across brand names, generic names, and manufacturer
- Health check endpoint with data freshness info
- Prometheus metrics endpoint
- Docker Compose deployment
- drug-cash integration as an upstream data source

### Out of Scope (v1)

- openFDA enrichment (adverse events, recalls, labels — drug-cash already caches these)
- Drug interaction data (separate data source, separate service)
- Historical NDC tracking (point-in-time snapshots)
- Write API (data comes exclusively from FDA bulk download)
- Authentication / API keys (internal network only, same as drug-cash)
- UI / dashboard

## 5. Architecture

### Tech Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| Language | Go or Python | Team's preference — Go for consistency with drug-cash, Python for faster dev with SQLAlchemy |
| Database | PostgreSQL 16+ | Relational data, full-text search via GIN indexes |
| HTTP Framework | Team's choice | net/http (Go) or FastAPI (Python) |
| Containers | Docker Compose | Match drug-cash deployment pattern |
| Metrics | Prometheus client | `/metrics` endpoint |

### Data Model

```sql
CREATE TABLE products (
    product_id          TEXT PRIMARY KEY,
    product_ndc         TEXT NOT NULL,
    product_type        TEXT,
    proprietary_name    TEXT,           -- brand name
    nonproprietary_name TEXT,           -- generic name
    dosage_form         TEXT,
    route               TEXT,
    labeler_name        TEXT,           -- manufacturer
    substance_name      TEXT,           -- active ingredients (semicolon-delimited)
    strength            TEXT,
    strength_unit       TEXT,
    pharm_classes       TEXT,           -- pharmacological classes
    dea_schedule        TEXT,
    marketing_category  TEXT,
    application_number  TEXT,
    marketing_start     DATE,
    marketing_end       DATE,
    ndc_exclude         BOOLEAN DEFAULT FALSE,
    listing_certified   DATE,
    search_vector       TSVECTOR        -- generated full-text search column
);

CREATE TABLE packages (
    id                  SERIAL PRIMARY KEY,
    product_id          TEXT NOT NULL REFERENCES products(product_id),
    product_ndc         TEXT NOT NULL,
    ndc_package_code    TEXT NOT NULL,   -- full 3-segment NDC
    description         TEXT,            -- "1 BOTTLE in 1 CARTON / 100 TABLET in 1 BOTTLE"
    marketing_start     DATE,
    marketing_end       DATE,
    ndc_exclude         BOOLEAN DEFAULT FALSE,
    sample_package      BOOLEAN DEFAULT FALSE
);

-- Indexes
CREATE INDEX idx_products_ndc ON products(product_ndc);
CREATE INDEX idx_products_name ON products USING GIN(search_vector);
CREATE INDEX idx_packages_product ON packages(product_id);
CREATE INDEX idx_packages_ndc ON packages(ndc_package_code);
CREATE INDEX idx_packages_product_ndc ON packages(product_ndc);
```

### Data Ingestion

```
Daily (cron or internal scheduler):
  1. Download https://www.accessdata.fda.gov/cder/ndctext.zip
  2. Unzip → product.txt + package.txt (tab-delimited)
  3. Parse and validate rows
  4. Load into staging tables (COPY FROM for speed)
  5. Swap staging → live (atomic rename or transaction)
  6. Update search vectors
  7. Record load metadata (row counts, duration, errors)
```

**Key requirement:** The swap must be atomic — consumers should never see partial data. Use a transaction that truncates + inserts, or use table renaming (`products_staging` → `products`).

**Error handling:** If download fails or row count drops > 20% from previous load, abort and keep existing data. Alert via Prometheus metric.

## 6. API Contract

Base URL: `http://ndc-loader:8081` (port configurable)

### `GET /api/ndc/{ndc}`

Look up a product by NDC code. Supports both 2-segment (product) and 3-segment (package) formats.

**Response:**
```json
{
  "product_ndc": "0002-1433",
  "brand_name": "Metformin Hydrochloride",
  "generic_name": "METFORMIN HYDROCHLORIDE",
  "dosage_form": "TABLET",
  "route": "ORAL",
  "manufacturer": "Eli Lilly and Company",
  "active_ingredients": [
    {"name": "METFORMIN HYDROCHLORIDE", "strength": "500", "unit": "mg/1"}
  ],
  "pharm_classes": ["Biguanide [EPC]", "Biguanide [Chemical/Ingredient]"],
  "dea_schedule": null,
  "marketing_category": "ANDA",
  "packages": [
    {
      "ndc": "0002-1433-02",
      "description": "100 TABLET in 1 BOTTLE"
    },
    {
      "ndc": "0002-1433-30",
      "description": "500 TABLET in 1 BOTTLE"
    }
  ]
}
```

**NDC format normalization:** Accept any common format — `0002-1433`, `00021433`, `0002-1433-02` — and normalize internally. If a 3-segment NDC is provided, return the parent product with all packages (highlight the matched package).

**Errors:**
- `404` — NDC not found in directory
- `400` — Invalid NDC format

### `GET /api/ndc/search?q={query}&limit=50&offset=0`

Full-text search across brand name, generic name, and manufacturer.

**Response:**
```json
{
  "query": "metformin",
  "total": 342,
  "limit": 50,
  "offset": 0,
  "results": [
    {
      "product_ndc": "0002-1433",
      "brand_name": "Metformin Hydrochloride",
      "generic_name": "METFORMIN HYDROCHLORIDE",
      "dosage_form": "TABLET",
      "manufacturer": "Eli Lilly and Company",
      "relevance": 0.95
    }
  ]
}
```

**Search behavior:**
- PostgreSQL `ts_query` with ranking by `ts_rank`
- Prefix matching supported (`metfor` matches `metformin`)
- Results ordered by relevance score descending

### `GET /api/ndc/{ndc}/packages`

List all packages for a product NDC.

**Response:**
```json
{
  "product_ndc": "0002-1433",
  "packages": [
    {
      "ndc": "0002-1433-02",
      "description": "100 TABLET in 1 BOTTLE",
      "sample": false
    }
  ]
}
```

### `GET /api/ndc/stats`

Dataset statistics for monitoring and debugging.

**Response:**
```json
{
  "products": 138742,
  "packages": 247891,
  "last_loaded": "2026-03-25T03:15:00Z",
  "load_duration_seconds": 42,
  "source": "https://www.accessdata.fda.gov/cder/ndctext.zip",
  "excluded_products": 12847
}
```

### `GET /health`

**Response:**
```json
{
  "status": "ok",
  "db": "connected",
  "data_age_hours": 4.2,
  "last_load": "2026-03-25T03:15:00Z"
}
```

**Degraded** if `data_age_hours > 48` (stale data warning).

### `GET /metrics`

Prometheus format. Key metrics:
- `ndc_loader_products_total` — current product count
- `ndc_loader_packages_total` — current package count
- `ndc_loader_load_duration_seconds` — last load duration
- `ndc_loader_load_last_success_timestamp` — unix timestamp of last successful load
- `ndc_loader_load_errors_total` — failed load attempts
- `ndc_loader_query_duration_seconds` — histogram of query latencies
- `ndc_loader_search_duration_seconds` — histogram of search latencies

## 7. drug-cash Integration

Once ndc-loader is running, drug-cash adds it as an upstream in `config.yaml`:

```yaml
- slug: ndc-products
  base_url: http://ndc-loader:8081
  path: /api/ndc/search
  format: json
  data_key: results
  total_key: total
  pagination_style: offset
  pagesize: 50
  search_params:
    - "q={QUERY}"
  ttl: "1h"

- slug: ndc-lookup
  base_url: http://ndc-loader:8081
  path: /api/ndc/{NDC}
  format: json
  ttl: "1h"
```

This gives drug-cash consumers:
- `GET /api/cache/ndc-products?QUERY=metformin` — search via drug-cash cache
- `GET /api/cache/ndc-lookup?NDC=0002-1433` — single NDC lookup via drug-cash cache
- Cross-slug search includes NDC data alongside DailyMed, FDA, and RxNorm results

The existing `fda-ndc` slug (openFDA API) can be deprecated once ndc-loader is validated.

## 8. Deployment

### Local Development
```
docker-compose up   # PostgreSQL + ndc-loader
```

### Production
Same deployment pattern as drug-cash:
- Docker Compose on self-hosted infrastructure
- Container registry: `dockerhub.calebdunn.tech/finish06/ndc-loader`
- Runs alongside drug-cash on the same Docker network

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://ndc:ndc@localhost:5432/ndc` | PostgreSQL connection string |
| `LISTEN_ADDR` | `:8081` | HTTP listen address |
| `LOAD_SCHEDULE` | `0 3 * * *` | Cron schedule for FDA data refresh |
| `FDA_NDC_URL` | `https://www.accessdata.fda.gov/cder/ndctext.zip` | Source URL |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |

## 9. Non-Functional Requirements

- **Performance:** Single NDC lookup < 5ms. Search < 50ms. Bulk load < 2 minutes.
- **Reliability:** Failed loads keep existing data. Never serve empty/partial dataset.
- **Observability:** Structured JSON logging. Prometheus metrics. Data freshness in health check.
- **Storage:** PostgreSQL with ~500MB disk for full dataset + indexes.
- **Security:** Internal network only. No auth required in v1. No PII in NDC data.

## 10. Open Questions

| # | Question | Context |
|---|----------|---------|
| 1 | Go or Python? | Go matches drug-cash stack. Python is faster to prototype with SQLAlchemy + FastAPI. Team preference? |
| 2 | Same host as drug-cash or separate? | Could share the Docker Compose or run on a different machine. Depends on resource constraints. |
| 3 | Should the loader expose a bulk export endpoint? | `GET /api/ndc/export?format=csv` for downstream analytics. Low effort if desired. |
| 4 | Include excluded NDCs (`NDC_EXCLUDE_FLAG=Y`)? | ~13K products. They're bulk ingredients, compounding components. Probably exclude by default, include via query param. |
| 5 | NDC format normalization depth? | FDA uses multiple formats (4-4-2, 5-3-2, 5-4-1). How aggressively should we normalize? Store canonical + accept any? |

## 11. Milestones (Suggested)

| Milestone | Goal | Effort |
|-----------|------|--------|
| M1: Schema + Loader | Download ZIP, parse, load into PostgreSQL, daily cron | 2-3 days |
| M2: Query API | `/ndc/{ndc}`, `/ndc/search`, `/ndc/{ndc}/packages` endpoints | 2-3 days |
| M3: Ops | Health check, Prometheus metrics, Docker Compose, structured logging | 1-2 days |
| M4: drug-cash Integration | Add ndc-loader as upstream in drug-cash config, validate cross-slug search | 1 day |
