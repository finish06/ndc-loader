# Milestone M1 — Schema + Loader

**Goal:** Download and ingest the FDA NDC Directory and Drugs@FDA datasets into PostgreSQL with checkpoint-based retry and API key authentication.
**Status:** IN_PROGRESS
**Appetite:** 1 week
**Target Maturity:** Alpha
**Started:** 2026-03-25

## Hill Chart

```
fda-data-fetcher    ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  uphill — spec complete, no implementation
```

## Features

| Feature | Spec | Position | Target | Notes |
|---------|------|----------|--------|-------|
| fda-data-fetcher | specs/fda-data-fetcher.md | SPECCED | VERIFIED | Download, parse, load FDA datasets with checkpoint retry + API key auth |

## Cycle Tracking

| Cycle | Features | Status | Notes |
|-------|----------|--------|-------|
| — | — | — | No cycles executed yet |

## Success Criteria

- [ ] NDC Directory products and packages tables populated from FDA bulk download
- [ ] Drugs@FDA tables (applications, products, submissions, marketing_status, active_ingredients, te_codes) populated
- [ ] Join works: products.application_number -> applications.appl_no
- [ ] Checkpoint-based retry: partial failures resume from last successful table
- [ ] Row count safety valve: abort if >20% drop
- [ ] API key authentication on all endpoints
- [ ] Atomic swap: consumers never see partial data
- [ ] Docker Compose runs full stack locally

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| FDA download URL changes | Load fails | Configurable URLs in datasets.yaml, discovery fallback |
| Drugs@FDA file format changes | Parse errors | Flexible column mapping, skip unknown columns |
| Large dataset load time | Slow dev cycle | Use subset for tests, full dataset for integration |
