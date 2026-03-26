# Milestone M1 — Schema + Loader

**Goal:** Download and ingest the FDA NDC Directory and Drugs@FDA datasets into PostgreSQL with checkpoint-based retry and API key authentication.
**Status:** IN_PROGRESS
**Appetite:** 1 week
**Target Maturity:** Alpha
**Started:** 2026-03-25

## Hill Chart

```
fda-data-fetcher    ██████████████████████████████████░░  downhill — implemented, tested, needs real FDA data validation
```

## Features

| Feature | Spec | Position | Target | Notes |
|---------|------|----------|--------|-------|
| fda-data-fetcher | specs/fda-data-fetcher.md | IN_PROGRESS | VERIFIED | All code + tests done. Needs E2E with real FDA data to reach VERIFIED. |

## Cycle Tracking

| Cycle | Features | Status | Notes |
|-------|----------|--------|-------|
| cycle-1 | fda-data-fetcher (SPECCED->IN_PROGRESS) | COMPLETE | 18/18 AC implemented, 90%+ coverage, 2 bugs fixed |

## Success Criteria

- [x] NDC Directory products and packages tables populated from FDA bulk download
- [x] Drugs@FDA tables (applications, products, submissions, marketing_status, active_ingredients, te_codes) populated
- [x] Join works: products.application_number -> applications.appl_no
- [x] Checkpoint-based retry: partial failures resume from last successful table
- [x] Row count safety valve: abort if >20% drop
- [x] API key authentication on all endpoints
- [x] Atomic swap: consumers never see partial data
- [x] Docker Compose runs full stack locally
- [ ] E2E validation: load real FDA data and verify row counts
- [ ] Query API endpoints (M2 scope)

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| FDA download URL changes | Load fails | Configurable URLs in datasets.yaml, discovery fallback |
| Drugs@FDA file format changes | Parse errors | Flexible column mapping, skip unknown columns |
| Large dataset load time | Slow dev cycle | Use subset for tests, full dataset for integration |
