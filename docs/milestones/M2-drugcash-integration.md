# Milestone M2 — drug-cash Integration

**Goal:** Make ndc-loader a drop-in upstream replacement for the openFDA NDC API in drug-cash. Consumers call drug-cash slugs, drug-cash proxies to ndc-loader, responses match openFDA format.
**Status:** IN_PROGRESS
**Appetite:** 3 days
**Target Maturity:** Alpha
**Started:** 2026-03-26

## Hill Chart

```
openfda-compat-api  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  not started
```

## Features

| Feature | Spec | Position | Target | Notes |
|---------|------|----------|--------|-------|
| openfda-compat-api | specs/openfda-compat-api.md | SHAPED | VERIFIED | openFDA-compatible response format for drug-cash drop-in |

## Cycle Tracking

| Cycle | Features | Status | Notes |
|-------|----------|--------|-------|
| — | — | — | No cycles executed yet |

## Success Criteria

- [ ] ndc-loader returns openFDA-compatible JSON for NDC lookup
- [ ] ndc-loader returns openFDA-compatible JSON for NDC search
- [ ] drug-cash config.yaml slugs point to ndc-loader
- [ ] drug-cash consumers get identical response shape to openFDA
- [ ] Existing fda-ndc slug in drug-cash can be deprecated
- [ ] Pagination matches openFDA meta format (skip/limit/total)
