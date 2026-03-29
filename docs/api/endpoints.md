# API Reference

Base URL: `http://localhost:8081`

All authenticated endpoints require the `X-API-Key` header.

---

## Query Endpoints

### GET /api/ndc/{ndc}

Look up a product by NDC code. Accepts any format — hyphenated, unhyphenated, 2-segment, 3-segment.

**Parameters:**
| Name | In | Type | Required | Description |
|------|----|------|----------|-------------|
| ndc | path | string | yes | NDC code (e.g., `0002-1433`, `0002-1433-61`, `00021433`) |

**Responses:**
- `200` — Product found with packages
- `400` — Invalid NDC format
- `401` — Missing or invalid API key
- `404` — NDC not found

**Example:**
```bash
curl http://localhost:8081/api/ndc/0002-1433 -H "X-API-Key: your-key"
```

```json
{
  "product_ndc": "0002-1433",
  "brand_name": "Trulicity",
  "generic_name": "DULAGLUTIDE",
  "dosage_form": "INJECTION, SOLUTION",
  "route": "SUBCUTANEOUS",
  "manufacturer": "Eli Lilly and Company",
  "active_ingredients": "DULAGLUTIDE",
  "strength": ".75",
  "strength_unit": "mg/.5mL",
  "pharm_classes": "GLP-1 Receptor Agonist [EPC], ...",
  "pharm_classes_structured": {
    "epc": ["GLP-1 Receptor Agonist"],
    "moa": ["Glucagon-like Peptide-1 (GLP-1) Agonists"],
    "cs": ["Glucagon-Like Peptide 1"],
    "pe": [],
    "raw": "GLP-1 Receptor Agonist [EPC], ..."
  },
  "dea_schedule": null,
  "marketing_category": "BLA",
  "application_number": "BLA125469",
  "packages": [
    {"ndc": "0002-1433-80", "description": "4 SYRINGE in 1 CARTON", "sample": false}
  ],
  "matched_package": null
}
```

When queried with a 3-segment NDC, `matched_package` is set to the matched package NDC.

---

### GET /api/ndc/search

Full-text search across brand names, generic names, and manufacturers.

**Parameters:**
| Name | In | Type | Required | Description |
|------|----|------|----------|-------------|
| q | query | string | yes | Search query (e.g., `metformin`) |
| limit | query | int | no | Results per page (default: 50, max: 100) |
| offset | query | int | no | Pagination offset (default: 0) |

**Response (200):**
```json
{
  "query": "metformin",
  "total": 586,
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

Supports prefix matching: `metfor` matches `metformin`.

---

### GET /api/ndc/{ndc}/packages

List all packages for a product NDC.

**Response (200):**
```json
{
  "product_ndc": "0002-1433",
  "packages": [
    {"ndc": "0002-1433-61", "description": "2 SYRINGE in 1 CARTON", "sample": true},
    {"ndc": "0002-1433-80", "description": "4 SYRINGE in 1 CARTON", "sample": false}
  ]
}
```

---

### GET /api/ndc/stats

Dataset statistics.

**Response (200):**
```json
{
  "products": 112230,
  "packages": 212309,
  "applications": 28959,
  "last_loaded": "2026-03-29T03:15:00Z",
  "load_duration_seconds": 6.5
}
```

---

## openFDA-Compatible Endpoint

### GET /api/openfda/ndc.json

Drop-in replacement for the openFDA `/drug/ndc.json` endpoint. Response format is identical to the real openFDA API.

**Parameters:**
| Name | In | Type | Required | Description |
|------|----|------|----------|-------------|
| search | query | string | yes | openFDA search query |
| limit | query | int | no | Results per page (default: 1, max: 1000) |
| skip | query | int | no | Pagination offset (default: 0) |

**Search syntax:**
- `brand_name:metformin` — field-specific search
- `brand_name:"Metformin Hydrochloride"` — exact phrase
- `metformin` — full-text (no field prefix)
- `brand_name:metformin+generic_name:hydrochloride` — AND

**Searchable fields:** `brand_name`, `generic_name`, `product_ndc`, `labeler_name`, `manufacturer_name`, `application_number`, `pharm_class`, `dosage_form`, `route`, `product_type`, `marketing_category`, `dea_schedule`

**Response (200):**
```json
{
  "meta": {
    "disclaimer": "This data is sourced from the FDA NDC Directory bulk download.",
    "terms": "https://open.fda.gov/terms/",
    "license": "https://open.fda.gov/license/",
    "last_updated": "2026-03-29",
    "results": {"skip": 0, "limit": 1, "total": 586}
  },
  "results": [
    {
      "product_ndc": "0002-1433",
      "brand_name": "Metformin Hydrochloride",
      "generic_name": "METFORMIN HYDROCHLORIDE",
      "active_ingredients": [{"name": "METFORMIN HYDROCHLORIDE", "strength": "500 mg/1"}],
      "packaging": [{"package_ndc": "0002-1433-61", "description": "...", "sample": false}],
      "openfda": {"manufacturer_name": ["Eli Lilly and Company"]},
      "route": ["ORAL"],
      "pharm_class": ["Biguanide [EPC]"],
      "...": "..."
    }
  ]
}
```

**Error (404):**
```json
{"error": {"code": "NOT_FOUND", "message": "No matches found!"}}
```

---

## Admin Endpoints

### POST /api/admin/load

Trigger a manual FDA data load.

**Request body (optional):**
```json
{"datasets": ["ndc_directory", "drugsfda"], "force": false}
```

**Response (202):**
```json
{"load_id": "550e8400-...", "status": "started", "datasets": ["ndc_directory", "drugsfda"]}
```

**Response (409):** Load already in progress.

### GET /api/admin/load/{loadID}

Check load status with per-table checkpoint details.

**Response (200):**
```json
{
  "load_id": "550e8400-...",
  "status": "in_progress",
  "started_at": "2026-03-29T03:00:00Z",
  "checkpoints": [
    {"dataset": "ndc_directory", "table": "products", "status": "loaded", "row_count": 112230, "duration_seconds": 2.1},
    {"dataset": "ndc_directory", "table": "packages", "status": "loading"}
  ]
}
```

---

## Operations Endpoints

### GET /health

Comprehensive health check with dependency status. **No auth required.**

**Response (200):**
```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime": "4h32m15s",
  "start_time": "2026-03-29T03:00:00Z",
  "data_age_hours": 4.2,
  "last_load": "2026-03-29T03:15:00Z",
  "dependencies": [
    {"name": "postgres", "status": "connected", "latency_ms": 1.2}
  ]
}
```

**Status values:**
- `ok` — postgres connected, data fresh (<48h)
- `degraded` — postgres connected, data stale (>48h) or never loaded
- `error` — postgres disconnected

### GET /version

Build and runtime metadata. **No auth required.**

**Response (200):**
```json
{
  "version": "0.1.0",
  "git_commit": "bf431aa",
  "git_branch": "main",
  "go_version": "go1.26.1",
  "os": "linux",
  "arch": "amd64",
  "build_time": "2026-03-29T14:55:00Z"
}
```

### GET /metrics

Prometheus metrics. **No auth required.** Key metrics:
- `ndc_loader_products_total` — current product count
- `ndc_loader_packages_total` — current package count
- `ndc_loader_load_duration_seconds` — last load duration
- `ndc_loader_load_errors_total` — failed load attempts
- `ndc_loader_query_duration_seconds` — query latency histogram

### GET /swagger/

Interactive API documentation (Swagger UI). **No auth required.**
