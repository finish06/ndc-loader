# Milestone M2 — drug-cash Integration

**Goal:** Make ndc-loader a drop-in upstream replacement for the openFDA NDC API in drug-cash.
**Status:** COMPLETE
**Appetite:** 3 days
**Target Maturity:** Alpha
**Started:** 2026-03-26
**Completed:** 2026-03-27

## Hill Chart

```
openfda-compat-api  ████████████████████████████████████  VERIFIED
```

## Features

| Feature | Spec | Position | Target | Notes |
|---------|------|----------|--------|-------|
| openfda-compat-api | specs/openfda-compat-api.md | VERIFIED | VERIFIED | 14/14 AC, format parity confirmed |

## Cycle Tracking

| Cycle | Features | Status | Notes |
|-------|----------|--------|-------|
| cycle-3 | openfda-compat-api (SHAPED->VERIFIED) | COMPLETE | Format parity verified against live openFDA API |

## Success Criteria

- [x] ndc-loader returns openFDA-compatible JSON for NDC lookup
- [x] ndc-loader returns openFDA-compatible JSON for NDC search
- [x] drug-cash config.yaml slugs documented (config lives in drug-cash repo)
- [x] drug-cash consumers get identical response shape to openFDA
- [x] Existing fda-ndc slug in drug-cash can be deprecated
- [x] Pagination matches openFDA meta format (skip/limit/total)
