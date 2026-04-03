# Cycle 5 — Landing Page + Release Tagging Spec

**Milestone:** M4 — Production Readiness
**Maturity:** Alpha
**Status:** PLANNED
**Started:** 2026-04-03
**Duration Budget:** 1 day

## Work Items

| Feature | Current Pos | Target Pos | Assigned | Est. Effort | Validation |
|---------|-------------|-----------|----------|-------------|------------|
| landing-page | SPECCED | VERIFIED | Agent-1 | ~4h | TDD cycle: redirect handler tests pass, landing page HTML complete, all 15 ACs met |
| release-tagging | SHAPED | SPECCED | Agent-1 | ~30min | Spec created via /add:spec interview, ACs defined |

## Dependencies & Serialization

```
landing-page (SPECCED → VERIFIED)
    ↓ (serial — complete before moving on)
release-tagging (SHAPED → SPECCED)
```

Single-threaded execution. Landing page is the primary deliverable; release tagging spec is a lightweight follow-up.

## Execution Plan

### Step 1: Landing Page TDD Cycle (~4h)

Follow `docs/plans/landing-page-plan.md`:

**Phase 1 — Root Redirect (Go changes, TDD):**
1. Add `LandingURL` field to `model.Config`
2. Read `LANDING_URL` env var in config loader
3. Write redirect handler + unit tests (RED → GREEN)
4. Register GET `/` in router (outside auth group)
5. Fix existing test call sites for NewRouter signature change
6. Add swagger annotation

**Phase 2 — Landing Page (Static HTML):**
1. Create `landing/index.html` — single self-contained file
2. Hero section: rx-dag branding, tagline, CTA buttons
3. Feature highlights: 4 cards (ingestion, lookup, search, openFDA compat)
4. API showcase: curl examples + sample JSON for key endpoints
5. Quick start: docker-compose setup, first API call
6. Footer: GitHub, Swagger, "Powered by ndc-loader"
7. Pharmacy green theme (#10b981, #0d9488), dark backgrounds, responsive

**Phase 3 — Verification:**
1. Full test suite passes (`go test ./...`)
2. Lint + vet clean
3. Open index.html in browser — visual check

### Step 2: Release Tagging Spec (~30min)

Run `/add:spec` interview for release tagging feature:
- Semantic versioning strategy
- Git tag workflow
- GitHub Actions release pipeline
- Changelog generation

## Blockers & Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Cloudflare DNS not configured | Cannot validate GitHub Pages deployment | Implementation proceeds; DNS is a manual step done before final verification. Flag as pending in cycle status. |
| NewRouter signature change | Breaks existing tests | TASK-009 in plan handles this — add empty string param to all call sites |

## Validation Criteria

### Per-Item Validation
- **landing-page:**
  - [ ] Redirect handler unit tests pass (4 test cases: with URL, without URL, empty string, no auth)
  - [ ] `landing/index.html` exists and renders all 5 sections
  - [ ] Pharmacy green theme applied
  - [ ] Responsive on mobile/tablet/desktop
  - [ ] Full `go test ./...` passes
  - [ ] `go vet ./...` clean
- **release-tagging:**
  - [ ] `specs/release-tagging.md` created with ACs and test cases

### Cycle Success Criteria
- [ ] Landing page feature at VERIFIED position
- [ ] Release tagging feature at SPECCED position
- [ ] All Go tests passing, no regressions
- [ ] No lint/vet errors

## Agent Autonomy

Alpha maturity, human available: interactive execution. Agent implements, human reviews as we go. Commits to feature branch, PR for review before merge.
