# Implementation Plan: rx-dag Landing Page & Root Redirect

**Spec:** specs/landing-page.md v0.1.0
**Created:** 2026-04-03
**Team Size:** Solo
**Estimated Duration:** 4-5 hours

## Overview

Two deliverables: (1) a self-contained static HTML landing page for the rx-dag public brand, deployed via GitHub Pages, and (2) a config-driven root redirect in the ndc-loader binary that sends GET `/` to either the landing page URL or `/swagger/`.

## Objectives

- Ship a polished public marketing page for rx-dag with zero build dependencies
- Add LANDING_URL env var support to the binary for configurable root redirect
- Keep the change minimal — no new Go dependencies, one new HTML file, small config/router changes

## Success Criteria

- All 15 acceptance criteria implemented and tested
- Root redirect unit tests pass
- Landing page renders correctly when opened in a browser
- All quality gates passing (lint, vet, tests)

## Acceptance Criteria Analysis

### AC-001 through AC-010: Static Landing Page
- **Complexity:** Medium (single file, but significant HTML/CSS craft)
- **Effort:** 2-3 hours
- **Tasks:** TASK-001 through TASK-003
- **Dependencies:** None — purely additive
- **Risk:** Low — no Go code, no integration points

### AC-011 through AC-013: Root Redirect
- **Complexity:** Simple (one handler, one config field, one route)
- **Effort:** 1 hour
- **Tasks:** TASK-004 through TASK-007
- **Dependencies:** Existing router and config patterns
- **Risk:** Low — follows established patterns exactly

### AC-014 through AC-015: Polish
- **Complexity:** Simple
- **Effort:** 30 min (bundled into TASK-001)
- **Dependencies:** Landing page exists

## Implementation Phases

### Phase 1: Root Redirect (Go changes) — TDD

The Go changes are small and testable. Do these first via TDD.

| Task ID | Description | Effort | AC | Dependencies |
|---------|-------------|--------|-----|--------------|
| TASK-004 | Add `LandingURL` field to `model.Config` struct | 5 min | AC-011 | — |
| TASK-005 | Read `LANDING_URL` env var in `internal/config.go` (empty string default) | 5 min | AC-011, AC-012 | TASK-004 |
| TASK-006 | Write redirect handler + unit tests (RED → GREEN) | 30 min | AC-011, AC-012, AC-013 | TASK-004 |
| TASK-007 | Register GET `/` route in `router.go` (outside auth group) and pass `LandingURL` from `main.go` | 15 min | AC-011, AC-012, AC-013 | TASK-005, TASK-006 |

**Files changed:**
- `internal/model/config.go` — add `LandingURL string` field
- `internal/config.go` — add `LANDING_URL` env var read
- `internal/api/router.go` — add `landingURL string` param to `NewRouter`, register GET `/`
- `internal/api/redirect.go` — new file, redirect handler function
- `internal/api/redirect_test.go` — new file, unit tests for redirect behavior
- `cmd/ndc-loader/main.go` — pass `cfg.LandingURL` to `NewRouter`

**Test cases:**
- `TestRootRedirect_WithLandingURL` — verify 302 to configured URL
- `TestRootRedirect_WithoutLandingURL` — verify 302 to `/swagger/`
- `TestRootRedirect_EmptyStringLandingURL` — verify 302 to `/swagger/`
- `TestRootRedirect_NoAuth` — verify no 401 when no API key provided

### Phase 2: Landing Page (Static HTML)

| Task ID | Description | Effort | AC | Dependencies |
|---------|-------------|--------|-----|--------------|
| TASK-001 | Create `docs/landing/index.html` with full page structure | 2h | AC-001 through AC-010, AC-014, AC-015 | — |
| TASK-002 | Add hero, features, API showcase, quick start, footer sections | (bundled) | AC-002 through AC-006 | TASK-001 |
| TASK-003 | Style with pharmacy green theme, responsive breakpoints, copy buttons | (bundled) | AC-007, AC-008, AC-010, AC-014 | TASK-001 |

**File created:**
- `docs/landing/index.html` — single self-contained file

**Content plan:**

```
Hero Section:
  - "rx-dag" in large type
  - Tagline: "Fast, local FDA drug data API"
  - Description: Replace openFDA dependency with a self-hosted NDC Directory API
  - CTA: "View API Docs" → /swagger/  |  "GitHub" → repo URL

Feature Cards (4):
  - FDA Data Ingestion: "112K+ products ingested daily from FDA bulk downloads"
  - NDC Lookup: "Any format, <5ms P95 — 4-4-2, 5-3-2, 5-4-1 all accepted"
  - Full-Text Search: "Prefix matching, relevance ranking across brand/generic/manufacturer"
  - openFDA Compatible: "Drop-in /drug/ndc.json replacement — zero code changes"

API Showcase (4 endpoints):
  - GET /api/ndc/0002-1433 — NDC lookup with sample response
  - GET /api/ndc/search?q=metformin — Search with sample response
  - GET /api/openfda/ndc.json?search=brand_name:Prozac — openFDA compat
  - GET /health — Health check

Quick Start:
  - docker-compose up
  - Set API_KEYS env var
  - First curl command

Footer:
  - Links: GitHub, Swagger, Health
  - "Powered by ndc-loader"
```

### Phase 3: Integration & Swagger Annotation

| Task ID | Description | Effort | AC | Dependencies |
|---------|-------------|--------|-----|--------------|
| TASK-008 | Add swagger annotation to redirect handler | 10 min | — | TASK-006 |
| TASK-009 | Update `NewRouter` call sites in existing tests (add `landingURL` param) | 15 min | — | TASK-007 |
| TASK-010 | Run full test suite, lint, vet | 10 min | — | All |

**Files changed:**
- `internal/api/redirect.go` — add swag comment block
- Any existing test files that call `NewRouter()` — add empty string param

## Effort Summary

| Phase | Tasks | Estimated Hours |
|-------|-------|-----------------|
| Phase 1: Root Redirect (TDD) | TASK-004 through TASK-007 | 1h |
| Phase 2: Landing Page | TASK-001 through TASK-003 | 2.5h |
| Phase 3: Integration | TASK-008 through TASK-010 | 0.5h |
| **Total** | **10 tasks** | **4h** |

## Task Execution Order

```
TASK-004 (Config struct)
  → TASK-005 (Config loader)
  → TASK-006 (Handler + tests — RED then GREEN)
  → TASK-007 (Router wiring)
  → TASK-009 (Fix existing test call sites)
  → TASK-008 (Swagger annotation)

TASK-001 (Landing page HTML — can start in parallel with Phase 1)
  → includes TASK-002 + TASK-003

TASK-010 (Final verification — last)
```

## Dependency Graph

```
No external dependencies.

Internal:
  TASK-006 depends on TASK-004 (needs Config type)
  TASK-007 depends on TASK-005 + TASK-006 (needs config value + handler)
  TASK-009 depends on TASK-007 (NewRouter signature change)
  TASK-001 is independent (static HTML, no Go code)
  TASK-010 depends on all others
```

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| NewRouter signature change breaks existing tests | Certain | Low | TASK-009 handles this — just add empty string param |
| GitHub Pages not enabled on repo | Low | Low | Manual config step — document in spec |
| Landing page curl examples become stale | Medium | Low | Examples use generic NDCs that exist in FDA data |

## Testing Strategy

### Unit Tests (Phase 1)
- `redirect_test.go`: 4 test functions covering AC-011, AC-012, AC-013, and empty-string edge case
- All use `httptest.NewRecorder` — no database or server needed

### Visual Verification (Phase 2)
- Open `docs/landing/index.html` directly in browser
- Check mobile viewport (375px), tablet (768px), desktop (1200px+)
- Verify all links point to correct targets

### Integration (Phase 3)
- Full `go test ./...` passes
- `go vet ./...` clean
- `golangci-lint run` clean

## Deliverables

### Code
- `internal/model/config.go` — add LandingURL field
- `internal/config.go` — read LANDING_URL env var
- `internal/api/redirect.go` — redirect handler
- `internal/api/redirect_test.go` — unit tests
- `internal/api/router.go` — register GET `/` route
- `cmd/ndc-loader/main.go` — pass LandingURL to router

### Static Assets
- `docs/landing/index.html` — complete landing page

### Documentation
- Swagger annotation on GET `/` endpoint

## Next Steps

1. Review and approve this plan
2. Run `/add:tdd-cycle specs/landing-page.md` to execute
3. Enable GitHub Pages on the repo pointing to `docs/landing/`
