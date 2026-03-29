# ndc-loader

FDA NDC Directory bulk loader and REST API. Downloads the complete NDC Directory and Drugs@FDA datasets daily from FDA bulk downloads, loads them into PostgreSQL, and serves them via a REST API with full-text search, NDC format normalization, and openFDA-compatible responses.

Replaces the openFDA API dependency for [drug-cash](https://github.com/finish06) and internal microservices.

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

# Browse interactive API docs
open http://localhost:8081/swagger/
```

## API

All endpoints require `X-API-Key` header unless noted. Interactive docs at `/swagger/`.

### Query Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/ndc/{ndc}` | Lookup by NDC code (any format) |
| `GET` | `/api/ndc/search?q={query}&limit=50&offset=0` | Full-text search |
| `GET` | `/api/ndc/{ndc}/packages` | List packages for a product |
| `GET` | `/api/ndc/stats` | Dataset statistics |

### openFDA-Compatible Endpoint

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/openfda/ndc.json?search={query}&limit=1&skip=0` | Drop-in replacement for openFDA `/drug/ndc.json` |

Supports openFDA search syntax: `brand_name:metformin`, `product_ndc:"0002-1433"`, AND via `+`.

### Admin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/admin/load` | Trigger manual data load |
| `GET` | `/api/admin/load/{id}` | Check load status with checkpoints |

### Operations (no auth required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check with postgres dep check, uptime, data freshness |
| `GET` | `/version` | Build info (git commit, branch, Go version, OS, arch) |
| `GET` | `/metrics` | Prometheus metrics |
| `GET` | `/swagger/` | Interactive API documentation (Swagger UI) |

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
  "brand_name": "Trulicity",
  "generic_name": "DULAGLUTIDE",
  "dosage_form": "INJECTION, SOLUTION",
  "route": "SUBCUTANEOUS",
  "manufacturer": "Eli Lilly and Company",
  "pharm_classes": "GLP-1 Receptor Agonist [EPC], Glucagon-Like Peptide 1 [CS]",
  "pharm_classes_structured": {
    "epc": ["GLP-1 Receptor Agonist"],
    "moa": ["Glucagon-like Peptide-1 (GLP-1) Agonists"],
    "cs": ["Glucagon-Like Peptide 1"],
    "pe": [],
    "raw": "GLP-1 Receptor Agonist [EPC], Glucagon-Like Peptide 1 [CS], ..."
  },
  "packages": [
    {"ndc": "0002-1433-80", "description": "4 SYRINGE in 1 CARTON", "sample": false}
  ]
}
```

## Data Sources

| Dataset | Source | Records | Refresh |
|---------|--------|---------|---------|
| NDC Directory | [FDA ndctext.zip](https://www.accessdata.fda.gov/cder/ndctext.zip) | ~112K products, ~212K packages | Daily 3am |
| Drugs@FDA | [FDA media/89850](https://www.fda.gov/media/89850/download) | ~29K applications, ~51K products, ~191K submissions | Daily 3am |

Datasets are joinable via `application_number` (NDC) to `appl_no` (Drugs@FDA) after stripping the type prefix (e.g., `ANDA076543` -> `076543`).

Additional datasets can be added via `datasets.yaml` without code changes.

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

## Development

```bash
make build                # Build binary with version info
make test                 # Unit tests
make test-integration     # Integration tests (requires postgres)
make test-e2e             # E2E tests (downloads real FDA data)
make lint                 # golangci-lint
make docs                 # Regenerate Swagger spec
make docker-build         # Build Docker image locally
```

## Deployment

### Staging (192.168.1.145)

```bash
make deploy-staging-first  # First time: sync config + .env template
make deploy-staging        # Routine: pull latest image + restart
make staging-status        # Health check
make staging-logs          # Tail logs
make staging-load          # Trigger FDA data load
make staging-psql          # Open psql shell
```

### CI/CD Pipeline

Push to `main` triggers: lint -> test -> build image -> push to registry -> staging webhook deploy.

Tag `v*` triggers: same pipeline + push `:version` and `:latest` tags.

```
push main → lint + test + vet → build :beta → push to registries → webhook → staging deploy
tag v*    → lint + test + vet → build :tag + :latest → push to registries
```

## Architecture

```
cmd/ndc-loader/         Entry point (ldflags: version, git commit, branch)
internal/
  api/                  HTTP handlers, middleware, NDC normalization, openFDA compat
  loader/               FDA download, parsing, orchestration, scheduling
  store/                PostgreSQL queries, bulk loading, checkpoints
  model/                Domain types
migrations/             SQL schema (embedded, auto-applied on startup)
datasets.yaml           Configurable dataset sources
docs/swagger/           Generated OpenAPI spec (served at /swagger/)
```

### System Overview

```mermaid
graph LR
    FDA[(FDA Bulk Downloads)] -->|Daily cron| Loader
    subgraph ndc-loader
        Loader[Data Loader] -->|COPY| PG[(PostgreSQL)]
        PG --> API[REST API]
        PG --> OpenFDA[openFDA Compat]
    end
    API -->|JSON| DC[drug-cash]
    OpenFDA -->|openFDA format| DC
    API -->|JSON| MS[Microservices]
    API -->|/metrics| Prom[Prometheus]
    API -->|/swagger/| Docs[Swagger UI]
```

### Data Pipeline

```mermaid
flowchart TD
    A[Cron Trigger / Manual POST] --> B[Download FDA ZIPs]
    B -->|Retry with backoff| B
    B -->|Success| C[Extract to temp dir]
    C --> D[Parse tab-delimited files]
    D -->|UTF-8 sanitize| D1[Sanitize invalid bytes]
    D1 --> E[Map columns + coerce types]
    E -->|Date: YYYYMMDD → time.Time| E
    E -->|Bool: Y/N → bool| E
    E --> F[Bulk COPY into staging table]
    F --> G{Row count safe?}
    G -->|Drop > 20%| H[Abort — keep existing data]
    G -->|OK| I[Atomic swap: staging → live]
    I --> J[Rebuild search vectors]
    J --> K[Record checkpoint metadata]
    K --> L{More tables?}
    L -->|Yes| D
    L -->|No| M[Load complete]

    style H fill:#ef4444,color:#fff
    style M fill:#22c55e,color:#fff
```

### Request Flow

```mermaid
flowchart LR
    Client -->|X-API-Key| MW[Auth Middleware]
    MW -->|401| Reject[Unauthorized]
    MW -->|Valid| Router

    Router --> Q1["GET /api/ndc/{ndc}"]
    Router --> Q2["GET /api/ndc/search"]
    Router --> Q3["GET /api/ndc/{ndc}/packages"]
    Router --> Q4["GET /api/ndc/stats"]
    Router --> OF["GET /api/openfda/ndc.json"]
    Router --> A1["POST /api/admin/load"]

    Q1 --> NDC[NDC Normalizer]
    NDC -->|2-segment| PL[Product Lookup]
    NDC -->|3-segment| PKL[Package Lookup]
    NDC -->|10-digit| VAR[Try 4-4-2 / 5-3-2 / 5-4-1]
    VAR --> PL
    VAR --> PKL

    PL --> DB[(PostgreSQL)]
    PKL --> DB
    Q2 -->|tsvector + ts_rank| DB
    Q3 --> DB
    Q4 --> DB
    OF -->|openFDA search parser| DB

    style Reject fill:#ef4444,color:#fff
```

### Data Model

```mermaid
erDiagram
    products ||--o{ packages : "product_ndc"
    products }o--o| applications : "application_number ~ appl_no"
    applications ||--o{ drugsfda_products : "appl_no"
    applications ||--o{ submissions : "appl_no"
    applications ||--o{ marketing_status : "appl_no"
    applications ||--o{ te_codes : "appl_no"

    products {
        text product_id PK
        text product_ndc
        text proprietary_name
        text nonproprietary_name
        text labeler_name
        text application_number
        tsvector search_vector
    }

    packages {
        serial id PK
        text product_ndc
        text ndc_package_code
        text description
    }

    applications {
        text appl_no PK
        text appl_type
        text sponsor_name
    }

    drugsfda_products {
        serial id PK
        text appl_no
        text drug_name
        text active_ingredient
    }

    submissions {
        serial id PK
        text appl_no
        text submission_type
        text submission_status
    }

    marketing_status {
        serial id PK
        text appl_no
        text marketing_status_id
    }

    te_codes {
        serial id PK
        text appl_no
        text te_code
    }
```

### Checkpoint & Recovery

```mermaid
stateDiagram-v2
    [*] --> Pending
    Pending --> Downloading
    Downloading --> Downloaded: Success
    Downloading --> Failed: Error (retry exhausted)
    Downloaded --> Loading
    Loading --> Loaded: Atomic swap OK
    Loading --> Failed: Row count drop / DB error
    Failed --> Downloading: Resume load

    note right of Failed
        On resume, skip tables
        with status = Loaded
    end note
```

### Resilience

- **Retry**: Downloads retry with exponential backoff (configurable max attempts)
- **Checkpoints**: Per-table progress tracking; resume from failure point
- **Row count safety**: Abort if row count drops >20% from previous load
- **Atomic swap**: Consumers never see partial data
- **UTF-8 sanitization**: Handles Windows-1252 bytes in FDA data

## drug-cash Integration

ndc-loader is a drop-in upstream replacement for the openFDA NDC API:

```yaml
# drug-cash config.yaml slugs:
- slug: ndc-products
  base_url: http://ndc-loader:8081
  path: /api/openfda/ndc.json
  search_params: ["search={QUERY}"]

- slug: ndc-lookup
  base_url: http://ndc-loader:8081
  path: /api/openfda/ndc.json
  search_params: ["search=product_ndc:\"{NDC}\"", "limit=1"]
```

## Tech Stack

- **Go 1.26+** with Chi v5 router
- **PostgreSQL 16+** with pgx v5 driver, GIN indexes for full-text search
- **Docker Compose** for local dev and deployment
- **Prometheus** metrics at `/metrics`
- **Swagger UI** at `/swagger/` (OpenAPI spec via swaggo/swag)
- **GitHub Actions** CI/CD with staging webhook deploy

## License

Internal use only.
