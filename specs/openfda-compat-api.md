# Spec: openFDA-Compatible API

**Version:** 0.1.0
**Created:** 2026-03-26
**PRD Reference:** ndc-loader-prd.md (Section 7 — drug-cash Integration)
**Status:** Done
**Completed:** 2026-03-27
**Milestone:** M2 — drug-cash Integration

## 1. Overview

Add openFDA-compatible API endpoints to ndc-loader so drug-cash can use it as a drop-in replacement for the openFDA `/drug/ndc.json` API. Response format must match the openFDA schema exactly so existing drug-cash consumers don't need code changes.

### User Story

As a **drug-cash consumer**, I want to query ndc-loader through drug-cash slugs and get responses in the same format as the openFDA API, so that switching from openFDA to ndc-loader requires zero client code changes.

## 2. Acceptance Criteria

| ID | Criterion | Priority |
|----|-----------|----------|
| AC-001 | GET /api/openfda/ndc.json?search={query}&limit={n}&skip={n} returns openFDA-format response | Must |
| AC-002 | Response includes `meta` object with disclaimer, terms, license, last_updated, results (skip/limit/total) | Must |
| AC-003 | Response `results` array contains product objects matching openFDA field names | Must |
| AC-004 | `active_ingredients` field is an array of `{name, strength}` objects (not a flat string) | Must |
| AC-005 | `packaging` field is an array of `{package_ndc, description, marketing_start_date, sample}` objects | Must |
| AC-006 | `route` field is an array of strings (not a single string) | Must |
| AC-007 | `pharm_class` field is an array of strings (split from semicolon-delimited source) | Must |
| AC-008 | `openfda` nested object is present (populate `manufacturer_name`, leave others as empty arrays) | Must |
| AC-009 | `search` parameter supports openFDA query syntax: `field:value`, `field:"exact phrase"` | Must |
| AC-010 | `search` parameter supports generic text search (no field prefix = full-text search) | Must |
| AC-011 | `limit` defaults to 1, max 1000 (matching openFDA behavior) | Must |
| AC-012 | `skip` defaults to 0 (matching openFDA pagination) | Must |
| AC-013 | 404 with openFDA error format when no results found | Must |
| AC-014 | All openFDA-compat endpoints require API key | Must |

## 3. API Contract

### GET /api/openfda/ndc.json

Mirrors the openFDA `/drug/ndc.json` endpoint.

**Query Parameters:**
- `search` — openFDA-style query (e.g., `brand_name:metformin`, `product_ndc:"0002-1433"`, or plain `metformin`)
- `limit` — results per page (default: 1, max: 1000)
- `skip` — offset for pagination (default: 0)

**Response (200):**
```json
{
  "meta": {
    "disclaimer": "This data is sourced from the FDA NDC Directory bulk download, not the openFDA API.",
    "terms": "https://open.fda.gov/terms/",
    "license": "https://open.fda.gov/license/",
    "last_updated": "2026-03-26",
    "results": {
      "skip": 0,
      "limit": 1,
      "total": 524
    }
  },
  "results": [
    {
      "product_ndc": "0002-1433",
      "generic_name": "METFORMIN HYDROCHLORIDE",
      "labeler_name": "Eli Lilly and Company",
      "brand_name": "Metformin Hydrochloride",
      "active_ingredients": [
        {"name": "METFORMIN HYDROCHLORIDE", "strength": "500 mg/1"}
      ],
      "finished": true,
      "packaging": [
        {
          "package_ndc": "0002-1433-61",
          "description": "100 TABLET in 1 BOTTLE",
          "marketing_start_date": "19950301",
          "sample": false
        }
      ],
      "openfda": {
        "manufacturer_name": ["Eli Lilly and Company"],
        "rxcui": [],
        "spl_set_id": [],
        "is_original_packager": [],
        "upc": [],
        "unii": []
      },
      "marketing_category": "ANDA",
      "dosage_form": "TABLET",
      "spl_id": "",
      "product_type": "HUMAN PRESCRIPTION DRUG",
      "route": ["ORAL"],
      "marketing_start_date": "19950301",
      "product_id": "0002-1433_b1c3d4e5",
      "application_number": "ANDA076543",
      "brand_name_base": "Metformin Hydrochloride",
      "pharm_class": ["Biguanide [EPC]"]
    }
  ]
}
```

**Response (404 — no results):**
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "No matches found!"
  }
}
```

## 4. Field Mapping (NDC DB -> openFDA format)

| openFDA Field | Source Table | Source Column | Transform |
|---------------|-------------|--------------|-----------|
| product_ndc | products | product_ndc | as-is |
| generic_name | products | nonproprietary_name | as-is |
| labeler_name | products | labeler_name | as-is |
| brand_name | products | proprietary_name | as-is |
| brand_name_base | products | proprietary_name | as-is (same as brand_name for our data) |
| active_ingredients | products | substance_name, strength, strength_unit | Parse semicolon-delimited into array of {name, strength} |
| finished | — | — | Always `true` |
| packaging | packages | ndc_package_code, description, marketing_start, sample_package | Join by product_ndc, format dates as YYYYMMDD |
| openfda.manufacturer_name | products | labeler_name | Wrap in array |
| marketing_category | products | marketing_category | as-is |
| dosage_form | products | dosage_form | as-is |
| product_type | products | product_type | as-is |
| route | products | route | Split on "; " into array |
| product_id | products | product_id | as-is |
| application_number | products | application_number | as-is |
| pharm_class | products | pharm_classes | Split on "; " into array |
| marketing_start_date | products | marketing_start | Format as YYYYMMDD string |
| spl_id | — | — | Empty string (not in our data) |

## 5. Search Parameter Parsing

The openFDA `search` parameter supports:
- `brand_name:metformin` — field-specific search
- `brand_name:"Metformin Hydrochloride"` — exact phrase match
- `metformin` — full-text search across all searchable fields
- `brand_name:metformin+generic_name:hydrochloride` — AND combination

Mapping openFDA field names to PostgreSQL columns:
| openFDA search field | DB column | Search method |
|---------------------|-----------|---------------|
| brand_name | proprietary_name | ILIKE or tsvector |
| generic_name | nonproprietary_name | ILIKE or tsvector |
| product_ndc | product_ndc | Exact match |
| manufacturer_name / labeler_name | labeler_name | ILIKE or tsvector |
| application_number | application_number | Exact match |
| pharm_class | pharm_classes | ILIKE |
| (no field prefix) | search_vector | Full-text tsvector |

## 6. drug-cash Configuration

Once ndc-loader openFDA-compat API is running, drug-cash adds these slugs:

```yaml
- slug: ndc-products
  base_url: http://ndc-loader:8081
  path: /api/openfda/ndc.json
  format: json
  data_key: results
  total_key: meta.results.total
  pagination_style: offset
  pagesize: 50
  search_params:
    - "search={QUERY}"
  ttl: "1h"

- slug: ndc-lookup
  base_url: http://ndc-loader:8081
  path: /api/openfda/ndc.json
  format: json
  data_key: results
  total_key: meta.results.total
  search_params:
    - "search=product_ndc:\"{NDC}\""
    - "limit=1"
  ttl: "1h"
```

## 7. Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-03-26 | 0.1.0 | calebdunn | Initial spec |
