# Milestone M1 — Schema + Loader

**Goal:** Download and ingest the FDA NDC Directory and Drugs@FDA datasets into PostgreSQL with checkpoint-based retry and API key authentication.
**Status:** COMPLETE
**Appetite:** 1 week
**Target Maturity:** Alpha
**Started:** 2026-03-25
**Completed:** 2026-03-26

## Hill Chart

```
fda-data-fetcher    ████████████████████████████████████  VERIFIED
query-api           ████████████████████████████████████  VERIFIED
```

## Features

| Feature | Spec | Position | Target | Notes |
|---------|------|----------|--------|-------|
| fda-data-fetcher | specs/fda-data-fetcher.md | VERIFIED | VERIFIED | 112K products, 212K packages, 29K applications loaded |
| query-api | specs/query-api.md | VERIFIED | VERIFIED | Lookup, search, packages, stats with NDC normalization |

## Cycle Tracking

| Cycle | Features | Status | Notes |
|-------|----------|--------|-------|
| cycle-1 | fda-data-fetcher (SPECCED->IN_PROGRESS) | COMPLETE | 18/18 AC, 90%+ coverage, 2 bugs fixed |
| cycle-2 | fda-data-fetcher E2E + query-api (IN_PROGRESS->VERIFIED) | COMPLETE | Real FDA data validated, 4 query endpoints, NDC normalization |

## Success Criteria

- [x] NDC Directory products and packages tables populated from FDA bulk download
- [x] Drugs@FDA tables populated
- [x] Join works: products.application_number -> applications.appl_no (59K matches)
- [x] Checkpoint-based retry: partial failures resume from last successful table
- [x] Row count safety valve: abort if >20% drop
- [x] API key authentication on all endpoints
- [x] Atomic swap: consumers never see partial data
- [x] Docker Compose runs full stack locally
- [x] E2E validation: real FDA data loaded and verified
- [x] Query API endpoints: lookup, search, packages, stats
