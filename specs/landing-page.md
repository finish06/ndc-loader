# Spec: rx-dag Landing Page & Root Redirect

**Version:** 0.1.0
**Created:** 2026-04-03
**PRD Reference:** docs/prd.md
**Status:** Draft

## 1. Overview

A public-facing marketing landing page for **rx-dag** (the consumer brand of ndc-loader), hosted on GitHub Pages. The page explains what the service does, showcases API endpoints with curl examples, and provides links to the GitHub repo, Swagger docs, and quick start instructions. The ndc-loader binary also gains a config-driven root redirect: when `LANDING_URL` is set, GET `/` returns an HTTP redirect to that URL; otherwise it redirects to `/swagger/`.

### User Story

As a **potential API consumer or developer**, I want to visit the rx-dag landing page and immediately understand what the service offers, see example API calls, and find links to docs and source code, so that I can evaluate and integrate rx-dag without reading source code.

## 2. Acceptance Criteria

| ID | Criterion | Priority |
|----|-----------|----------|
| AC-001 | A single self-contained `index.html` exists at `docs/landing/index.html` with all CSS/JS inlined | Must |
| AC-002 | Landing page includes a hero section with rx-dag name, tagline, and brief description | Must |
| AC-003 | Landing page includes a feature highlights section covering: FDA data ingestion, NDC lookup, full-text search, openFDA compatibility | Must |
| AC-004 | Landing page includes an API endpoint showcase with example curl commands and sample JSON responses for key endpoints | Must |
| AC-005 | Landing page includes a quick start section with setup/usage instructions | Must |
| AC-006 | Landing page includes links to: GitHub repository, Swagger UI, health endpoint | Must |
| AC-007 | Landing page uses a pharmacy/medical green color scheme (#10b981, #0d9488) with dark backgrounds | Must |
| AC-008 | Landing page is fully static — no live API calls, no external JS dependencies, no build step | Must |
| AC-009 | Landing page is deployable via GitHub Pages from `docs/landing/` directory | Must |
| AC-010 | Landing page is responsive and renders correctly on mobile, tablet, and desktop viewports | Should |
| AC-011 | When `LANDING_URL` env var is set, GET `/` returns HTTP 302 redirect to the configured URL | Must |
| AC-012 | When `LANDING_URL` env var is not set or empty, GET `/` returns HTTP 302 redirect to `/swagger/` | Must |
| AC-013 | The root redirect route requires no authentication (no API key) | Must |
| AC-014 | Curl example code blocks have a visual copy affordance (copy button or select-all styling) | Should |
| AC-015 | Landing page footer includes "Powered by ndc-loader" with link to repo | Nice |
| AC-016 | GitHub Actions workflow deploys `docs/landing/` to GitHub Pages automatically on push to main | Must |

## 3. User Test Cases

### TC-001: View Landing Page

**Precondition:** Landing page deployed to GitHub Pages (or opened locally)
**Steps:**
1. Navigate to the rx-dag GitHub Pages URL
2. Observe the hero section loads with rx-dag branding
3. Scroll to feature highlights section
4. Scroll to API endpoint showcase
5. Scroll to quick start section
6. Verify all external links (GitHub, Swagger) are present
**Expected Result:** All sections render correctly with green/teal theme, no broken layouts, all links present
**Screenshot Checkpoint:** `tests/screenshots/landing-page/step-01-full-page.png`
**Maps to:** TBD

### TC-002: Mobile Responsiveness

**Precondition:** Landing page deployed or opened locally
**Steps:**
1. Open landing page in a mobile viewport (375px width)
2. Scroll through all sections
3. Verify no horizontal overflow, text is readable, code blocks scroll horizontally
**Expected Result:** Page is fully usable on mobile — no content cut off, navigation works
**Screenshot Checkpoint:** `tests/screenshots/landing-page/step-02-mobile.png`
**Maps to:** TBD

### TC-003: Root Redirect with LANDING_URL Set

**Precondition:** ndc-loader running with `LANDING_URL=https://rx-dag.example.com`
**Steps:**
1. Send GET request to `/`
2. Observe HTTP response
**Expected Result:** HTTP 302 with `Location: https://rx-dag.example.com`
**Screenshot Checkpoint:** N/A (API test)
**Maps to:** TBD

### TC-004: Root Redirect without LANDING_URL

**Precondition:** ndc-loader running without `LANDING_URL` set
**Steps:**
1. Send GET request to `/`
2. Observe HTTP response
**Expected Result:** HTTP 302 with `Location: /swagger/`
**Screenshot Checkpoint:** N/A (API test)
**Maps to:** TBD

### TC-005: Root Redirect Requires No Auth

**Precondition:** ndc-loader running with API key authentication enabled
**Steps:**
1. Send GET request to `/` with no `X-API-Key` header
2. Observe HTTP response
**Expected Result:** HTTP 302 redirect (not 401 Unauthorized)
**Screenshot Checkpoint:** N/A (API test)
**Maps to:** TBD

## 4. Data Model

No new data entities. This feature is a static page and a route configuration.

### Configuration

| Field | Source | Required | Description |
|-------|--------|----------|-------------|
| LANDING_URL | Environment variable | No | Full URL to redirect GET `/` to. If empty/unset, redirects to `/swagger/` |

## 5. API Contract

### GET /

**Description:** Root redirect — sends visitors to the landing page or Swagger UI.

**Request:** No parameters, no authentication required.

**Response (302 — LANDING_URL set):**
```
HTTP/1.1 302 Found
Location: https://rx-dag.example.com
```

**Response (302 — LANDING_URL not set):**
```
HTTP/1.1 302 Found
Location: /swagger/
```

## 6. Landing Page Structure

### Sections (top to bottom)

1. **Hero** — rx-dag logo/name, tagline (e.g., "Fast, local FDA drug data API"), brief 1-2 sentence description, CTA buttons (View Docs, GitHub)
2. **Feature Highlights** — 4 cards:
   - FDA Data Ingestion (daily bulk download, 112K+ products)
   - NDC Lookup (any format, <5ms P95)
   - Full-Text Search (prefix matching, relevance ranking)
   - openFDA Compatible (drop-in replacement, zero code changes)
3. **API Endpoint Showcase** — 3-4 key endpoints with curl examples and sample JSON responses:
   - `GET /api/ndc/{ndc}` — NDC lookup
   - `GET /api/ndc/search?q=metformin` — Full-text search
   - `GET /api/openfda/ndc.json?search=...` — openFDA compatibility
   - `GET /health` — Health check
4. **Quick Start** — Docker compose setup, env var configuration, first API call
5. **Links/Footer** — GitHub repo, Swagger UI URL, license info, "Powered by ndc-loader"

### Color Scheme

| Element | Color | Usage |
|---------|-------|-------|
| Background (primary) | #0f172a | Page background, dark navy |
| Background (cards) | #1e293b | Card backgrounds, code blocks |
| Accent (primary) | #10b981 | Headlines, buttons, highlights |
| Accent (secondary) | #0d9488 | Secondary elements, hover states |
| Accent (gradient) | #10b981 → #0d9488 | CTA buttons, hero accent |
| Text (primary) | #f1f5f9 | Body text |
| Text (secondary) | #94a3b8 | Muted text, descriptions |
| Code background | #0f172a | Inline code, code blocks |
| Border | #334155 | Card borders, dividers |

## 7. Edge Cases

| Case | Expected Behavior |
|------|-------------------|
| LANDING_URL set to empty string | Treat as unset — redirect to `/swagger/` |
| LANDING_URL set to invalid URL | Redirect anyway — the browser handles the invalid location |
| LANDING_URL has trailing slash | Use as-is, do not strip or add slashes |
| User visits landing page with JS disabled | Page must be fully readable without JavaScript (CSS-only layout) |
| Very long API response examples | Code blocks scroll horizontally, don't break layout |

## 8. Dependencies

- GitHub Pages enabled on the repository (deploy from `docs/landing/` directory)
- Existing Swagger UI at `/swagger/` (already implemented, specs/swagger-docs.md — Done)
- No new Go dependencies required for the redirect (uses standard `net/http`)

## 9. Revision History

| Date | Version | Author | Changes |
|------|---------|--------|---------|
| 2026-04-03 | 0.1.0 | calebdunn | Initial spec from /add:spec interview |
