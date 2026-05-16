# Architecture Diagrams

Generated from `.add/docs-manifest.json`. Source of truth is the Go code under `internal/`; if a diagram conflicts with the code, the code wins — regenerate via `/add:docs --scope diagrams`.

## Contents

- [System Overview](#system-overview)
- [Request Lifecycle](#request-lifecycle)
- [Authentication Middleware](#authentication-middleware)
- [Query Routing & NDC Normalization](#query-routing--ndc-normalization)
- [openFDA Compatibility Flow](#openfda-compatibility-flow)
- [Admin Load Trigger & Status](#admin-load-trigger--status)
- [Data Load Pipeline](#data-load-pipeline)
- [Checkpoint State Machine](#checkpoint-state-machine)
- [Resume After Failure](#resume-after-failure)
- [Health Check Flow](#health-check-flow)
- [Data Model (ER)](#data-model-er)

## System Overview

```mermaid
flowchart LR
    FDA[(FDA Bulk Downloads<br/>ndctext.zip + drugsatfda.zip)]
    Cron[Cron Scheduler<br/>LOAD_SCHEDULE]
    subgraph ndc-loader
        Orchestrator
        API[Chi Router<br/>:8081]
        DB[(PostgreSQL 16<br/>products / packages /<br/>applications / drugsfda_*)]
    end
    DrugCash[drug-cash]
    Microservices[Internal microservices]
    Prom[Prometheus]
    Swagger[Swagger UI]

    Cron --> Orchestrator
    Orchestrator -->|HTTPS| FDA
    FDA -->|ZIP| Orchestrator
    Orchestrator -->|COPY + atomic swap| DB
    API --> DB
    API -->|openFDA-shaped JSON| DrugCash
    API -->|JSON| Microservices
    API -->|/metrics| Prom
    API -->|/swagger/*| Swagger
```

## Request Lifecycle

End-to-end happy path for an authenticated query request.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant MW as Chi middleware<br/>(RequestID, RealIP,<br/>Recoverer, Compress)
    participant Auth as APIKeyAuth
    participant H as Handler
    participant Q as QueryStore
    participant DB as PostgreSQL

    C->>MW: GET /api/ndc/0002-1433<br/>X-API-Key: ***
    MW->>Auth: forward request
    alt missing or invalid key
        Auth-->>C: 401 unauthorized
    else valid key
        Auth->>H: forward
        H->>H: ParseNDC + NDCSearchVariants
        H->>Q: LookupByProductNDC(variants)
        Q->>DB: SELECT products WHERE product_ndc = $1
        DB-->>Q: ProductResult
        Q-->>H: ProductResult
        H->>H: enrichProduct (parse pharm_classes)
        H-->>C: 200 JSON
    end
```

## Authentication Middleware

```mermaid
flowchart TD
    Req[Incoming request] --> Header{X-API-Key header<br/>present?}
    Header -->|No| Deny[401 unauthorized<br/>+ JSON error body]
    Header -->|Yes| Set{Key in<br/>configured allowlist?}
    Set -->|No| Deny
    Set -->|Yes| Pass[next.ServeHTTP]

    style Deny fill:#ef4444,color:#fff
    style Pass fill:#22c55e,color:#fff
```

Configured via `APIKeyAuth([]string)` in `internal/api/middleware.go`. Operations endpoints (`/`, `/health`, `/version`, `/metrics`, `/swagger/*`) bypass this group.

## Query Routing & NDC Normalization

```mermaid
flowchart LR
    R[Chi router] --> Q1["GET /api/ndc/{ndc}"]
    R --> Q2[GET /api/ndc/search]
    R --> Q3["GET /api/ndc/{ndc}/packages"]
    R --> Q4[GET /api/ndc/stats]

    Q1 --> P[ParseNDC]
    Q3 --> P
    P -->|3-segment| PKG[LookupByPackageNDC]
    P -->|2-segment| PRD[LookupByProductNDC]
    P -->|10-digit| VAR[NDCSearchVariants<br/>4-4-2 / 5-3-2 / 5-4-1]
    VAR --> PKG
    VAR --> PRD

    Q2 --> FTS[SearchProducts<br/>tsvector + ts_rank]
    Q4 --> ST[GetStats]

    PKG --> DB[(PostgreSQL)]
    PRD --> DB
    FTS --> DB
    ST --> DB
```

## openFDA Compatibility Flow

```mermaid
sequenceDiagram
    autonumber
    participant C as drug-cash / client
    participant H as OpenFDAHandler.HandleNDCJSON
    participant P as ParseOpenFDASearch
    participant B as BuildSearchQuery
    participant Q as QueryStore.OpenFDASearch
    participant T as TransformToOpenFDA
    participant DB as PostgreSQL

    C->>H: GET /api/openfda/ndc.json?search=brand_name:metformin
    alt search param missing
        H-->>C: 400 OpenFDAError{BAD_REQUEST}
    else
        H->>P: parse search syntax<br/>(field:value, "exact", +AND)
        P-->>H: clauses
        H->>B: build WHERE clause + args
        B-->>H: (whereClause, args)
        H->>Q: OpenFDASearch(where, args, limit, skip)
        Q->>DB: parameterized SELECT
        DB-->>Q: products + total
        Q-->>H: ([]ProductResult, total)
        alt no matches
            H-->>C: 404 OpenFDAError{NOT_FOUND}
        else
            loop per product
                H->>T: TransformToOpenFDA(product)
                T-->>H: OpenFDAProduct
            end
            H-->>C: 200 OpenFDAResponse{meta, results}
        end
    end
```

## Admin Load Trigger & Status

```mermaid
sequenceDiagram
    autonumber
    participant Op as Operator
    participant Trig as AdminHandler.TriggerLoad
    participant Orc as Orchestrator
    participant CP as CheckpointStore
    participant Stat as AdminHandler.GetLoadStatus

    Op->>Trig: POST /api/admin/load { datasets?, force? }
    alt Orchestrator.GetActiveLoadID != ""
        Trig-->>Op: 409 { error: load_in_progress, load_id }
    else
        Trig->>Orc: go RunLoad(ctx, datasets, force, "")
        Trig-->>Op: 202 { load_id, status: started }
    end

    Note over Orc: load runs asynchronously (see Data Load Pipeline)

    Op->>Stat: GET /api/admin/load/{loadID}
    Stat->>CP: GetCheckpoints(loadID)
    alt no rows
        Stat-->>Op: 404 { error: load_not_found }
    else
        Stat-->>Op: 200 { status, started_at, checkpoints[] }
    end
```

## Data Load Pipeline

```mermaid
flowchart TD
    Start[RunLoad: assign load_id<br/>set activeLoadID]
    Start --> Enum[EnabledDatasets +<br/>optional name filter]
    Enum --> Loop{For each dataset}
    Loop --> DL[Downloader.Download<br/>retry w/ exponential backoff]
    DL --> EX[Downloader.Extract<br/>to temp dir]
    EX --> FileLoop{For each file}
    FileLoop --> CP1[CreateCheckpoint<br/>status=pending]
    CP1 --> Parse[ParseTabDelimited<br/>UTF-8 sanitize, LazyQuotes]
    Parse --> Map[MapColumns via<br/>headerMappings]
    Map --> Safe{force?}
    Safe -->|No| Prev[GetPreviousRowCount]
    Prev --> Drop{Row drop<br/>> threshold?}
    Drop -->|Yes| Abort[Abort table<br/>SetError]
    Drop -->|No| BL
    Safe -->|Yes| BL[DataLoader.BulkLoad<br/>COPY into staging]
    BL --> Swap[Atomic swap:<br/>staging → live<br/>inside transaction]
    Swap --> CP2[SetRowCount<br/>status=loaded]
    CP2 --> FileLoop
    FileLoop -->|done| Cleanup[Cleanup extract dir]
    Cleanup --> Loop
    Loop -->|done| End[clear activeLoadID]

    style Abort fill:#ef4444,color:#fff
    style End fill:#22c55e,color:#fff
```

## Checkpoint State Machine

Per-table checkpoint lifecycle, defined in `internal/model/config.go` (`LoadStatus` constants) and driven by `CheckpointStore.UpdateStatus`.

```mermaid
stateDiagram-v2
    [*] --> pending: CreateCheckpoint
    pending --> downloading: UpdateStatus
    downloading --> downloaded: download success
    downloading --> failed: retries exhausted
    downloaded --> loading: parse + map start
    loading --> loaded: atomic swap OK
    loading --> failed: safety check fail / DB error
    failed --> downloading: resume w/ existing load_id

    note right of failed
        ErrorMessage populated.
        On resume, GetLoadedTables
        skips status=loaded rows.
    end note
```

## Resume After Failure

```mermaid
sequenceDiagram
    autonumber
    participant Op as Operator
    participant Orc as Orchestrator
    participant CP as CheckpointStore

    Op->>Orc: RunLoad(ctx, datasets, force, resumeLoadID)
    Orc->>CP: GetLoadedTables(resumeLoadID)
    CP-->>Orc: map[table]bool (status=loaded)
    loop per dataset / file
        alt table already loaded
            Orc->>Orc: skip
        else
            Orc->>Orc: download + parse + map + bulk load
        end
    end
```

## Health Check Flow

`/health` is unauthenticated. Status degrades when the data is older than 48h or when no data has loaded yet; it errors only if PostgreSQL is unreachable.

```mermaid
flowchart TD
    Hit[GET /health] --> PG[checkPostgres:<br/>db.Ping with latency]
    PG --> Q{Ping ok?}
    Q -->|No| ErrSet[status = error<br/>dep status = disconnected]
    Q -->|Yes| OK[dep status = connected]
    OK --> Age[GetLastLoadInfo:<br/>last_load + age_hours]
    Age --> AgeQ{lastLoad nil?}
    AgeQ -->|Yes| Deg1[status = degraded<br/>no data loaded]
    AgeQ -->|No| Old{age > 48h?}
    Old -->|Yes| Deg2[status = degraded<br/>data stale]
    Old -->|No| Ok2[status = ok]
    Deg1 --> Resp[HealthResponse JSON]
    Deg2 --> Resp
    Ok2 --> Resp
    ErrSet --> Resp

    style ErrSet fill:#ef4444,color:#fff
    style Deg1 fill:#eab308,color:#000
    style Deg2 fill:#eab308,color:#000
    style Ok2 fill:#22c55e,color:#fff
```

## Data Model (ER)

```mermaid
erDiagram
    products ||--o{ packages : "product_ndc"
    products }o--o| applications : "application_number ~ appl_no"
    applications ||--o{ drugsfda_products : "appl_no"
    applications ||--o{ submissions : "appl_no"
    applications ||--o{ marketing_status : "appl_no"
    applications ||--o{ te_codes : "appl_no"
    load_checkpoints {
        text load_id
        text dataset
        text table_name
        text status
        int row_count
    }

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
        text product_id
        text product_ndc
        text ndc_package_code
        text description
        bool sample_package
    }
    applications {
        text appl_no PK
        text appl_type
        text sponsor_name
    }
    drugsfda_products {
        serial id PK
        text appl_no
        text product_no
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

Join across datasets uses `products.application_number` (`ANDA076543`) → `applications.appl_no` (`076543`) after stripping the `NDA/ANDA/BLA` prefix.

---

*Last updated: 2026-05-16 — covers 12 routes, 5 middleware layers, 7 base tables + checkpoint table.*
