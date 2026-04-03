# Spec: Swagger API Documentation

**Version:** 0.1.0
**Created:** 2026-03-27
**PRD Reference:** docs/prd.md
**Status:** Done
**Completed:** 2026-04-01
**Milestone:** M3 — Swagger Docs

## 1. Overview

Auto-generated OpenAPI 3.0 documentation for all ndc-loader API endpoints, served at runtime via Swagger UI. Uses `swaggo/swag` to generate the spec from Go code annotations. Consumers can explore and test the API interactively from a browser.

### User Story

As a **drug-cash developer or API consumer**, I want to browse interactive API documentation at `/swagger/`, so that I can discover endpoints, understand request/response formats, and test queries without reading source code.

## 2. Acceptance Criteria

| ID | Criterion | Priority |
|----|-----------|----------|
| AC-001 | `swag init` generates valid OpenAPI 3.0 spec from Go annotations | Must |
| AC-002 | Swagger UI served at `/swagger/` (no auth required) | Must |
| AC-003 | All query endpoints documented: GET /api/ndc/{ndc}, /api/ndc/search, /api/ndc/{ndc}/packages, /api/ndc/stats | Must |
| AC-004 | Admin endpoints documented: POST /api/admin/load, GET /api/admin/load/{id} | Must |
| AC-005 | openFDA-compat endpoint documented: GET /api/openfda/ndc.json | Must |
| AC-006 | Health and metrics endpoints documented: GET /health, GET /metrics | Must |
| AC-007 | Request parameters documented (path params, query params, headers) | Must |
| AC-008 | Response schemas documented with example JSON for each endpoint | Must |
| AC-009 | Error responses documented (400, 401, 404, 500) | Must |
| AC-010 | API key auth scheme documented (X-API-Key header, securityDefinitions) | Must |
| AC-011 | Swagger spec auto-regenerated via `go generate` or Makefile target | Should |
| AC-012 | API title, version, description, contact info populated from config | Should |

## 3. User Test Cases

### TC-001: Browse Swagger UI

**Precondition:** ndc-loader running locally
**Steps:**
1. Open `http://localhost:8081/swagger/` in browser
2. Swagger UI loads with API title "ndc-loader"
3. All endpoint groups visible (Query, Admin, OpenFDA, Operations)
**Expected Result:** Interactive documentation renders with all endpoints listed.
**Maps to:** TBD

### TC-002: Try Out NDC Lookup

**Precondition:** Swagger UI loaded, data loaded in DB
**Steps:**
1. Expand GET /api/ndc/{ndc}
2. Click "Try it out"
3. Enter NDC "0002-1433" and API key
4. Click "Execute"
**Expected Result:** 200 response with product JSON shown inline.
**Maps to:** TBD

### TC-003: View Error Response

**Precondition:** Swagger UI loaded
**Steps:**
1. Expand GET /api/ndc/{ndc}
2. View "Responses" section
3. Check 400, 401, 404 documented
**Expected Result:** Each error code shows description and example response body.
**Maps to:** TBD

## 4. Implementation Notes

### swaggo/swag Setup

```go
// cmd/ndc-loader/main.go — top-level annotations
// @title ndc-loader API
// @version 1.0
// @description FDA NDC Directory bulk loader and REST API
// @host localhost:8081
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
```

Each handler gets `@Summary`, `@Description`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Router` annotations.

### Dependencies

- `github.com/swaggo/swag` — annotation parser + spec generator
- `github.com/swaggo/http-swagger` — Swagger UI middleware for Chi
- Generated `docs/` package (committed to repo for Docker builds)

### Route Registration

```go
import httpSwagger "github.com/swaggo/http-swagger"

// No auth required.
r.Get("/swagger/*", httpSwagger.WrapHandler)
```

## 5. Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-03-27 | 0.1.0 | calebdunn | Initial spec |
