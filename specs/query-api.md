# Spec: Query API

**Version:** 0.1.0
**Created:** 2026-03-26
**PRD Reference:** ndc-loader-prd.md (Section 6)
**Status:** Draft

## 1. Overview

REST API for querying the loaded FDA NDC Directory and Drugs@FDA data. Supports single NDC lookup with format normalization, full-text search across drug names, package enumeration, and dataset statistics.

### User Story

As a **drug-cash consumer**, I want to query the ndc-loader API by NDC code or search term, so that I can look up drug information without depending on the openFDA API.

## 2. Acceptance Criteria

| ID | Criterion | Priority |
|----|-----------|----------|
| AC-001 | GET /api/ndc/{ndc} returns product details with packages for a valid NDC | Must |
| AC-002 | NDC lookup accepts any format: hyphenated (0002-1433), unhyphenated (00021433), 3-segment (0002-1433-02) | Must |
| AC-003 | NDC normalization handles all FDA segment patterns: 4-4-2, 5-3-2, 5-4-1 | Must |
| AC-004 | 3-segment NDC lookup returns parent product with matched package highlighted | Must |
| AC-005 | GET /api/ndc/search returns full-text search results with relevance ranking | Must |
| AC-006 | Search supports prefix matching (metfor -> metformin) | Must |
| AC-007 | Search returns paginated results with total count | Must |
| AC-008 | GET /api/ndc/{ndc}/packages returns all packages for a product NDC | Must |
| AC-009 | GET /api/ndc/stats returns dataset statistics (counts, last load, freshness) | Must |
| AC-010 | All query endpoints require API key authentication | Must |
| AC-011 | 404 returned for NDC not found | Must |
| AC-012 | 400 returned for invalid NDC format | Must |
| AC-013 | Query latency < 5ms P95 for single NDC lookup | Should |
| AC-014 | Search latency < 50ms P95 | Should |

## 3. NDC Format Normalization

FDA uses three segment patterns for 10-digit NDC codes:
- **4-4-2**: labeler(4)-product(4)-package(2) — e.g., 0002-1433-02
- **5-3-2**: labeler(5)-product(3)-package(2) — e.g., 12345-678-90
- **5-4-1**: labeler(5)-product(4)-package(1) — e.g., 12345-6789-0

Input formats accepted:
- Hyphenated 2-segment: "0002-1433" → product NDC lookup
- Hyphenated 3-segment: "0002-1433-02" → package NDC lookup
- Unhyphenated 10-digit: "0002143302" → infer segments, lookup as package
- Unhyphenated shorter: "00021433" → infer as product NDC

## 4. API Contract

### GET /api/ndc/{ndc}

**Response (200):**
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
  "dea_schedule": null,
  "marketing_category": "ANDA",
  "application_number": "ANDA076543",
  "packages": [
    {"ndc": "0002-1433-02", "description": "100 TABLET in 1 BOTTLE"},
    {"ndc": "0002-1433-30", "description": "500 TABLET in 1 BOTTLE"}
  ],
  "matched_package": null
}
```

When queried with a 3-segment NDC, `matched_package` is set to the matching NDC.

### GET /api/ndc/search?q={query}&limit=50&offset=0

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

### GET /api/ndc/{ndc}/packages

**Response (200):**
```json
{
  "product_ndc": "0002-1433",
  "packages": [
    {"ndc": "0002-1433-02", "description": "100 TABLET in 1 BOTTLE", "sample": false}
  ]
}
```

### GET /api/ndc/stats

**Response (200):**
```json
{
  "products": 112230,
  "packages": 212309,
  "applications": 28959,
  "last_loaded": "2026-03-26T22:21:53Z",
  "load_duration_seconds": 6.5,
  "source": "https://www.accessdata.fda.gov/cder/ndctext.zip"
}
```
