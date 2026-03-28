# Milestone M3 — Swagger API Documentation

**Goal:** Auto-generated OpenAPI 3.0 docs served via Swagger UI at /swagger/ for all API endpoints.
**Status:** IN_PROGRESS
**Appetite:** 1 day
**Target Maturity:** Alpha
**Started:** 2026-03-27

## Hill Chart

```
swagger-docs  ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  uphill — spec complete, not started
```

## Features

| Feature | Spec | Position | Target | Notes |
|---------|------|----------|--------|-------|
| swagger-docs | specs/swagger-docs.md | SPECCED | VERIFIED | swaggo/swag annotations, Swagger UI, all endpoints |

## Cycle Tracking

| Cycle | Features | Status | Notes |
|-------|----------|--------|-------|
| cycle-4 | swagger-docs (SPECCED->VERIFIED) | PLANNED | ~4 hours estimated |

## Success Criteria

- [ ] Swagger UI loads at /swagger/ (no auth)
- [ ] All endpoints documented with params, responses, examples
- [ ] API key auth scheme documented
- [ ] Error responses documented
- [ ] swag init regenerates cleanly
