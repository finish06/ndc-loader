# ndc-loader

FDA NDC Directory bulk loader and REST API. Downloads the complete NDC Directory and Drugs@FDA datasets daily from FDA bulk downloads, loads them into PostgreSQL, and serves them via a REST API with full-text search and NDC format normalization.

Replaces the openFDA API dependency for [drug-cash](https://github.com/calebdunn) and internal microservices.

## Quick Start

```bash
# Clone and start
cp .env.example .env
# Edit .env — set API_KEYS to a real secret
docker compose up -d

# Trigger initial data load
curl -X POST http://localhost:8081/api/admin/load \
  -H "X-API-Key: your-secret-key-here" \
  -H "Content-Type: application/json" \
  -d '{}'

# Search for a drug
curl http://localhost:8081/api/ndc/search?q=metformin \
  -H "X-API-Key: your-secret-key-here"
```

## API

All endpoints require `X-API-Key` header (except `/health` and `/metrics`).

### Query Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/ndc/{ndc}` | Lookup by NDC code (any format) |
| `GET` | `/api/ndc/search?q={query}&limit=50&offset=0` | Full-text search |
| `GET` | `/api/ndc/{ndc}/packages` | List packages for a product |
| `GET` | `/api/ndc/stats` | Dataset statistics |

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/admin/load` | Trigger manual data load |
| `GET` | `/api/admin/load/{id}` | Check load status |

### Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check with data freshness |
| `GET` | `/metrics` | Prometheus metrics |

### NDC Format Normalization

Accepts any common NDC format:
- Hyphenated 2-segment: `0002-1433` (product lookup)
- Hyphenated 3-segment: `0002-1433-61` (package lookup, returns parent product)
- Unhyphenated 10-digit: `0002143361` (tries 4-4-2, 5-3-2, 5-4-1 patterns)
- Unhyphenated shorter: `00021433` (product lookup)

### Example Response

```json
{
  "product_ndc": "0002-1433",
  "brand_name": "Metformin Hydrochloride",
  "generic_name": "METFORMIN HYDROCHLORIDE",
  "dosage_form": "TABLET",
  "route": "ORAL",
  "manufacturer": "Eli Lilly and Company",
  "active_ingredients": "METFORMIN HYDROCHLORIDE",
  "strength": "500",
  "strength_unit": "mg/1",
  "pharm_classes": "Biguanide [EPC]",
  "marketing_category": "ANDA",
  "application_number": "ANDA076543",
  "packages": [
    {"ndc": "0002-1433-61", "description": "100 TABLET in 1 BOTTLE", "sample": false}
  ],
  "matched_package": null
}
```

## Data Sources

| Dataset | Source | Records | Refresh |
|---------|--------|---------|---------|
| NDC Directory | [FDA ndctext.zip](https://www.accessdata.fda.gov/cder/ndctext.zip) | ~112K products, ~212K packages | Daily |
| Drugs@FDA | [FDA media/89850](https://www.fda.gov/media/89850/download) | ~29K applications, ~51K products, ~191K submissions | Daily |

Datasets are joinable via `application_number` (NDC) to `appl_no` (Drugs@FDA) after stripping the type prefix (e.g., `ANDA076543` -> `076543`).

## Configuration

Copy `.env.example` to `.env` and configure:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://ndc:ndc@localhost:5432/ndc` | PostgreSQL connection |
| `API_KEYS` | (required) | Comma-separated API keys |
| `LISTEN_ADDR` | `:8081` | HTTP listen address |
| `LOAD_SCHEDULE` | `0 3 * * *` | Cron schedule for daily refresh |
| `POSTGRES_PORT` | `5432` | Host port for PostgreSQL |
| `APP_PORT` | `8081` | Host port for ndc-loader |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |

Additional datasets can be added via `datasets.yaml` without code changes.

## Development

```bash
# Run unit tests
go test -race -cover ./internal/...

# Run integration tests (requires PostgreSQL)
docker compose up -d postgres
DATABASE_URL="postgres://ndc:ndc@localhost:5435/ndc?sslmode=disable" \
  go test -tags=integration ./tests/integration/...

# Run E2E tests (downloads real FDA data)
DATABASE_URL="postgres://ndc:ndc@localhost:5435/ndc?sslmode=disable" \
  go test -tags=e2e -timeout=10m ./tests/e2e/...

# Lint
golangci-lint run ./...
```

## Architecture

```
cmd/ndc-loader/         Entry point
internal/
  api/                  HTTP handlers, middleware, NDC normalization
  loader/               FDA download, parsing, orchestration, scheduling
  store/                PostgreSQL queries, bulk loading, checkpoints
  model/                Domain types
migrations/             SQL schema (embedded, auto-applied on startup)
datasets.yaml           Configurable dataset sources
```

### Data Pipeline

```
Cron trigger (daily 3am)
  -> Download FDA ZIP files (with retry + exponential backoff)
  -> Extract to temp directory
  -> Parse tab-delimited files (with UTF-8 sanitization)
  -> Map columns to DB schema (date/boolean coercion)
  -> Bulk COPY into staging tables
  -> Atomic swap (staging -> live)
  -> Update search vectors (tsvector trigger)
  -> Record checkpoint metadata
```

### Resilience

- **Retry**: Downloads retry with exponential backoff (configurable max attempts)
- **Checkpoints**: Per-table progress tracking; resume from failure point
- **Row count safety**: Abort if row count drops >20% from previous load
- **Atomic swap**: Consumers never see partial data
- **UTF-8 sanitization**: Handles Windows-1252 bytes in FDA data

## Tech Stack

- **Go 1.26+** with Chi v5 router
- **PostgreSQL 16+** with pgx v5 driver
- **Docker Compose** for local dev and deployment
- **Prometheus** metrics at `/metrics`

## License

Internal use only.
