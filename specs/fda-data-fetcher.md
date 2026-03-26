# Spec: FDA Data Fetcher

**Version:** 0.1.0
**Created:** 2026-03-25
**PRD Reference:** docs/prd.md, ndc-loader-prd.md
**Status:** Draft

## 1. Overview

A configurable FDA bulk data download and ingestion system. Discovers available datasets from the openFDA downloads page, downloads configured datasets (NDC Directory and Drugs@FDA initially), parses their contents, and loads them into PostgreSQL with separate tables per dataset — joinable via `application_number`.

The system supports checkpoint-based retry so that partial failures resume from the last successful step rather than restarting from scratch.

### User Story

As an **internal service operator**, I want ndc-loader to automatically download and ingest multiple FDA bulk datasets into a local PostgreSQL database, so that drug-cash and other consumers can query drug data without depending on external APIs.

## 2. Acceptance Criteria

| ID | Criterion | Priority |
|----|-----------|----------|
| AC-001 | System can fetch the openFDA downloads page and parse available dataset URLs | Must |
| AC-002 | System downloads the NDC Directory ZIP (ndctext.zip) from FDA | Must |
| AC-003 | System downloads all Drugs@FDA bulk data files from FDA | Must |
| AC-004 | Downloaded ZIPs are extracted and tab-delimited files parsed correctly | Must |
| AC-005 | NDC Directory data is loaded into `products` and `packages` tables | Must |
| AC-006 | Drugs@FDA data is loaded into separate tables (applications, products, submissions, marketing_status, active_ingredients, te_codes) | Must |
| AC-007 | All Drugs@FDA tables are joinable to NDC products via `application_number` | Must |
| AC-008 | Data load uses atomic swap — consumers never see partial data | Must |
| AC-009 | Checkpoint system tracks progress per-table so retries resume from last failure point | Must |
| AC-010 | If download fails, system retries with exponential backoff (max 3 attempts) | Must |
| AC-011 | If table N of M fails to load, tables 1..N-1 remain loaded; retry starts at table N | Must |
| AC-012 | If row count drops >20% from previous load for any table, abort that table's load and keep existing data | Must |
| AC-013 | Dataset list is configurable — new FDA datasets can be added without code changes | Should |
| AC-014 | API key authentication via `X-API-Key` header on all API endpoints | Must |
| AC-015 | Requests without valid API key receive 401 Unauthorized | Must |
| AC-016 | API keys are configurable via environment variable | Must |
| AC-017 | Load metadata is recorded (row counts, duration, errors, checkpoint state) per dataset | Must |
| AC-018 | Daily scheduled execution via cron or internal scheduler | Must |

## 3. User Test Cases

### TC-001: Full Fresh Load — NDC Directory

**Precondition:** Empty database, FDA download URL accessible
**Steps:**
1. Trigger data load (manual or scheduled)
2. System fetches openFDA downloads page
3. System downloads ndctext.zip
4. System extracts product.txt and package.txt
5. System parses tab-delimited rows
6. System loads into `products` and `packages` tables via atomic swap
**Expected Result:** Both tables populated. Row counts match source files. Load metadata recorded with duration and counts.
**Screenshot Checkpoint:** N/A (backend service)
**Maps to:** TBD

### TC-002: Full Fresh Load — Drugs@FDA

**Precondition:** Empty database, FDA download URL accessible
**Steps:**
1. Trigger data load
2. System downloads Drugs@FDA bulk files
3. System extracts and parses all Drugs@FDA data files
4. System loads into separate tables (applications, drugsfda_products, submissions, marketing_status, active_ingredients, te_codes)
**Expected Result:** All Drugs@FDA tables populated. Application numbers present for join with NDC products.
**Screenshot Checkpoint:** N/A
**Maps to:** TBD

### TC-003: Join Across Datasets

**Precondition:** Both NDC Directory and Drugs@FDA loaded
**Steps:**
1. Query a product from `products` table that has an `application_number`
2. Join to `applications` table on `application_number` = `appl_no`
3. Verify linked data (sponsor name, application type, approval dates)
**Expected Result:** Join returns matching application record with correct sponsor and approval data.
**Screenshot Checkpoint:** N/A
**Maps to:** TBD

### TC-004: Checkpoint Recovery — Partial Table Failure

**Precondition:** NDC products loaded successfully, packages load fails mid-way (simulated)
**Steps:**
1. Start a full load
2. Products table loads successfully — checkpoint recorded
3. Packages load fails at row N (simulate via injected error)
4. System records failure checkpoint
5. Retry triggered
6. System skips products (already checkpointed), resumes packages load
**Expected Result:** Products table untouched. Packages table retried from scratch (atomic swap means per-table retry, not per-row). Checkpoint log shows skip of completed tables.
**Screenshot Checkpoint:** N/A
**Maps to:** TBD

### TC-005: Download Failure with Retry

**Precondition:** FDA URL temporarily unreachable
**Steps:**
1. Trigger data load
2. First download attempt fails (network error)
3. System waits (exponential backoff) and retries
4. Second attempt succeeds
5. Load continues normally
**Expected Result:** Data loaded successfully after retry. Retry attempts logged. Metrics updated.
**Screenshot Checkpoint:** N/A
**Maps to:** TBD

### TC-006: Row Count Safety Valve

**Precondition:** Previous load had 140K products. New source file has 100K (>20% drop).
**Steps:**
1. Trigger data load
2. System downloads and parses new data
3. System detects row count dropped from 140K to 100K (28.5% drop)
4. System aborts load for that table
**Expected Result:** Existing data preserved. Error logged with expected vs actual counts. Prometheus metric incremented.
**Screenshot Checkpoint:** N/A
**Maps to:** TBD

### TC-007: API Key Authentication — Valid Key

**Precondition:** API running, valid API key configured
**Steps:**
1. Send GET /api/ndc/search?q=metformin with `X-API-Key: {valid_key}` header
**Expected Result:** 200 OK with search results.
**Screenshot Checkpoint:** N/A
**Maps to:** TBD

### TC-008: API Key Authentication — Missing/Invalid Key

**Precondition:** API running
**Steps:**
1. Send GET /api/ndc/search?q=metformin without API key header
2. Send GET /api/ndc/search?q=metformin with `X-API-Key: wrong-key`
**Expected Result:** Both return 401 Unauthorized with `{"error": "unauthorized", "message": "valid API key required"}`.
**Screenshot Checkpoint:** N/A
**Maps to:** TBD

### TC-009: Discover Available Datasets

**Precondition:** openFDA downloads page accessible
**Steps:**
1. System fetches https://open.fda.gov/data/downloads/
2. System parses page to extract available dataset categories and download URLs
3. System compares discovered datasets against configured allowlist
4. System downloads only configured datasets
**Expected Result:** Discovery log shows all available datasets. Only configured datasets (NDC Directory, Drugs@FDA) are downloaded.
**Screenshot Checkpoint:** N/A
**Maps to:** TBD

## 4. Data Model

### NDC Directory Tables

#### products

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| product_id | TEXT (PK) | Yes | FDA product ID |
| product_ndc | TEXT | Yes | 2-segment NDC (labeler-product) |
| product_type | TEXT | No | HUMAN PRESCRIPTION DRUG, OTC, etc. |
| proprietary_name | TEXT | No | Brand name |
| nonproprietary_name | TEXT | No | Generic name |
| dosage_form | TEXT | No | TABLET, CAPSULE, etc. |
| route | TEXT | No | ORAL, TOPICAL, etc. |
| labeler_name | TEXT | No | Manufacturer |
| substance_name | TEXT | No | Active ingredients (semicolon-delimited) |
| strength | TEXT | No | Strength values |
| strength_unit | TEXT | No | Strength units |
| pharm_classes | TEXT | No | Pharmacological classes |
| dea_schedule | TEXT | No | DEA schedule (CII-CV) |
| marketing_category | TEXT | No | NDA, ANDA, BLA, OTC, etc. |
| application_number | TEXT | No | NDA/ANDA number — **JOIN KEY to Drugs@FDA** |
| marketing_start | DATE | No | Marketing start date |
| marketing_end | DATE | No | Marketing end date |
| ndc_exclude | BOOLEAN | No | Excluded from NDC directory |
| listing_certified | DATE | No | Listing certification date |
| search_vector | TSVECTOR | No | Generated full-text search column |

#### packages

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | SERIAL (PK) | Yes | Auto-increment ID |
| product_id | TEXT (FK) | Yes | References products.product_id |
| product_ndc | TEXT | Yes | Parent product NDC |
| ndc_package_code | TEXT | Yes | Full 3-segment NDC |
| description | TEXT | No | Package description |
| marketing_start | DATE | No | Marketing start date |
| marketing_end | DATE | No | Marketing end date |
| ndc_exclude | BOOLEAN | No | Excluded flag |
| sample_package | BOOLEAN | No | Sample package flag |

### Drugs@FDA Tables

#### applications

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| appl_no | TEXT (PK) | Yes | Application number — **JOIN KEY to NDC products** |
| appl_type | TEXT | No | NDA, ANDA, BLA |
| sponsor_name | TEXT | No | Sponsor/company name |
| most_recent_submission | DATE | No | Most recent submission date |

#### drugsfda_products

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | SERIAL (PK) | Yes | Auto-increment |
| appl_no | TEXT (FK) | Yes | References applications.appl_no |
| product_no | TEXT | Yes | Product number within application |
| form | TEXT | No | Dosage form |
| strength | TEXT | No | Strength |
| reference_drug | TEXT | No | Reference drug flag |
| drug_name | TEXT | No | Drug name |
| active_ingredient | TEXT | No | Active ingredient |
| reference_standard | TEXT | No | Reference standard flag |

#### submissions

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | SERIAL (PK) | Yes | Auto-increment |
| appl_no | TEXT (FK) | Yes | References applications.appl_no |
| submission_type | TEXT | No | ORIG, SUPPL, etc. |
| submission_no | TEXT | No | Submission number |
| submission_status | TEXT | No | AP (approved), TA, etc. |
| submission_status_date | DATE | No | Status date |
| submission_class_code | TEXT | No | Classification code |
| submission_class_code_description | TEXT | No | Code description |

#### marketing_status

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | SERIAL (PK) | Yes | Auto-increment |
| appl_no | TEXT (FK) | Yes | References applications.appl_no |
| product_no | TEXT | No | Product number |
| marketing_status_id | TEXT | No | Status ID |
| marketing_status | TEXT | No | Status description |

#### active_ingredients

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | SERIAL (PK) | Yes | Auto-increment |
| appl_no | TEXT (FK) | Yes | References applications.appl_no |
| product_no | TEXT | No | Product number |
| ingredient_name | TEXT | No | Active ingredient name |
| strength | TEXT | No | Strength with units |

#### te_codes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | SERIAL (PK) | Yes | Auto-increment |
| appl_no | TEXT (FK) | Yes | References applications.appl_no |
| product_no | TEXT | No | Product number |
| te_code | TEXT | No | Therapeutic equivalence code |

### Load Metadata

#### load_checkpoints

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | SERIAL (PK) | Yes | Auto-increment |
| load_id | UUID | Yes | Unique load execution ID |
| dataset | TEXT | Yes | Dataset name (ndc_directory, drugsfda) |
| table_name | TEXT | Yes | Target table name |
| status | TEXT | Yes | pending, downloading, downloaded, loading, loaded, failed |
| row_count | INTEGER | No | Rows loaded |
| previous_row_count | INTEGER | No | Previous load row count (for safety valve) |
| error_message | TEXT | No | Error details if failed |
| started_at | TIMESTAMPTZ | No | Step start time |
| completed_at | TIMESTAMPTZ | No | Step completion time |
| created_at | TIMESTAMPTZ | Yes | Record creation time |

### Relationships

```
products.application_number ──────> applications.appl_no
packages.product_id ──────────────> products.product_id
drugsfda_products.appl_no ────────> applications.appl_no
submissions.appl_no ──────────────> applications.appl_no
marketing_status.appl_no ─────────> applications.appl_no
active_ingredients.appl_no ───────> applications.appl_no
te_codes.appl_no ─────────────────> applications.appl_no
```

### Indexes

```sql
-- NDC Directory
CREATE INDEX idx_products_ndc ON products(product_ndc);
CREATE INDEX idx_products_appno ON products(application_number);
CREATE INDEX idx_products_search ON products USING GIN(search_vector);
CREATE INDEX idx_packages_product ON packages(product_id);
CREATE INDEX idx_packages_ndc ON packages(ndc_package_code);
CREATE INDEX idx_packages_product_ndc ON packages(product_ndc);

-- Drugs@FDA
CREATE INDEX idx_drugsfda_products_appno ON drugsfda_products(appl_no);
CREATE INDEX idx_submissions_appno ON submissions(appl_no);
CREATE INDEX idx_marketing_status_appno ON marketing_status(appl_no);
CREATE INDEX idx_active_ingredients_appno ON active_ingredients(appl_no);
CREATE INDEX idx_te_codes_appno ON te_codes(appl_no);

-- Load tracking
CREATE INDEX idx_checkpoints_load ON load_checkpoints(load_id);
CREATE INDEX idx_checkpoints_dataset ON load_checkpoints(dataset, status);
```

## 5. API Contract

_Note: API endpoints for querying this data are covered in a separate spec (query-api). This spec covers the data fetching, ingestion, and storage layer. The API key middleware defined here applies to all endpoints._

### Authentication Middleware

All API endpoints require `X-API-Key` header.

**Request Header:**
```
X-API-Key: {configured_api_key}
```

**Failure Response (401):**
```json
{
  "error": "unauthorized",
  "message": "valid API key required"
}
```

API keys configured via `API_KEYS` environment variable (comma-separated for multiple keys).

### POST /api/admin/load

Trigger a manual data load (for testing or recovery).

**Request:**
```json
{
  "datasets": ["ndc_directory", "drugsfda"],
  "force": false
}
```

- `datasets`: which datasets to load (default: all configured)
- `force`: skip row count safety valve (default: false)

**Response (202 Accepted):**
```json
{
  "load_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "started",
  "datasets": ["ndc_directory", "drugsfda"]
}
```

**Response (409 Conflict):**
```json
{
  "error": "load_in_progress",
  "load_id": "existing-load-id"
}
```

### GET /api/admin/load/{load_id}

Check status of a load operation.

**Response (200):**
```json
{
  "load_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "in_progress",
  "started_at": "2026-03-25T03:00:00Z",
  "checkpoints": [
    {"dataset": "ndc_directory", "table": "products", "status": "loaded", "row_count": 138742, "duration_seconds": 18},
    {"dataset": "ndc_directory", "table": "packages", "status": "loading", "row_count": null},
    {"dataset": "drugsfda", "table": "applications", "status": "pending"}
  ]
}
```

## 6. Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://ndc:ndc@localhost:5432/ndc` | PostgreSQL connection string |
| `API_KEYS` | (required) | Comma-separated valid API keys |
| `LOAD_SCHEDULE` | `0 3 * * *` | Cron schedule for data refresh |
| `FDA_DOWNLOADS_URL` | `https://open.fda.gov/data/downloads/` | openFDA downloads page |
| `DOWNLOAD_DIR` | `/tmp/fda-data` | Temporary directory for downloads |
| `MAX_RETRY_ATTEMPTS` | `3` | Max download retry attempts |
| `ROW_COUNT_DROP_THRESHOLD` | `0.20` | Max allowed row count drop (20%) |

### Dataset Configuration (datasets.yaml)

```yaml
datasets:
  - name: ndc_directory
    enabled: true
    source_url: https://www.accessdata.fda.gov/cder/ndctext.zip
    format: zip
    files:
      - filename: product.txt
        table: products
        delimiter: "\t"
        has_header: true
      - filename: package.txt
        table: packages
        delimiter: "\t"
        has_header: true

  - name: drugsfda
    enabled: true
    source_url: https://www.fda.gov/media/89850/download
    format: zip
    files:
      - filename: Applications.txt
        table: applications
        delimiter: "\t"
        has_header: true
      - filename: Products.txt
        table: drugsfda_products
        delimiter: "\t"
        has_header: true
      - filename: Submissions.txt
        table: submissions
        delimiter: "\t"
        has_header: true
      - filename: MarketingStatus.txt
        table: marketing_status
        delimiter: "\t"
        has_header: true
      - filename: ActionTypes_Lookup.txt
        table: active_ingredients
        delimiter: "\t"
        has_header: true
      - filename: TECodes.txt
        table: te_codes
        delimiter: "\t"
        has_header: true
```

## 7. Edge Cases

| Case | Expected Behavior |
|------|-------------------|
| FDA download URL returns 404 | Retry up to MAX_RETRY_ATTEMPTS, then fail with error logged. Existing data preserved. |
| FDA download URL returns 503 | Retry with exponential backoff. |
| ZIP file corrupted | Detect via ZIP integrity check. Fail, log error, preserve existing data. |
| Tab-delimited file has unexpected columns | Log warning, map known columns, ignore unknown. Do not abort. |
| Tab-delimited file has fewer columns than expected | Log warning per row, skip malformed rows, continue. |
| Row count drops >20% | Abort load for that table. Existing data preserved. Alert via metrics. |
| application_number is NULL on NDC product | Valid — not all NDC products have FDA applications. Join simply returns no match. |
| Duplicate application_number across datasets | Expected — multiple NDC products can reference the same application. This is a many-to-one relationship. |
| Concurrent load triggered while one is running | Return 409 Conflict. Only one load runs at a time. |
| Database connection lost mid-load | Transaction rollback. Checkpoint records failure. Retry resumes from failed table. |
| Disk full during download | Fail gracefully, clean up partial files, log error. |
| openFDA downloads page changes HTML structure | Discovery fails gracefully. Fall back to configured source_url per dataset. Log warning. |

## 8. Dependencies

- PostgreSQL 16+ (via Docker Compose for local dev)
- pgx v5 (Go PostgreSQL driver)
- Chi v5 (HTTP router)
- `net/http` (for FDA downloads)
- `archive/zip` (stdlib, for ZIP extraction)
- `encoding/csv` (stdlib, for tab-delimited parsing)
- `robfig/cron` or similar (for scheduled execution)
- Prometheus client library (for metrics)

## 9. Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-03-25 | 0.1.0 | calebdunn | Initial spec from /add:spec interview |
