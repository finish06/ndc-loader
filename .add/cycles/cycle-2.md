# Cycle 2 — E2E FDA Validation + Query API

**Milestone:** M1 (close) + M2 (start)
**Maturity:** Alpha
**Status:** PLANNED
**Started:** TBD
**Completed:** TBD
**Duration Budget:** 3 days (autonomous — human mostly away)

## Work Items

| Feature | Current Pos | Target Pos | Assigned | Est. Effort | Validation |
|---------|-------------|-----------|----------|-------------|------------|
| fda-data-fetcher (E2E) | IN_PROGRESS | VERIFIED | Agent-1 | ~3 hours | Real FDA data loaded, row counts verified, joins confirmed |
| query-api | SHAPED | VERIFIED | Agent-1 | ~15 hours | All query endpoints working, NDC normalization, full-text search, 90%+ coverage |

## Dependencies & Serialization

```
fda-data-fetcher E2E validation (must complete first — query API needs data in DB)
    ↓
query-api (depends on loaded data for integration tests)
```

Single-threaded execution. E2E validation first, then query API.

## Implementation Order

### Phase 1: E2E FDA Data Validation (~3 hours)
- [ ] Download real NDC Directory from FDA (subset: first 1000 rows for speed)
- [ ] Download real Drugs@FDA from FDA (subset: first 1000 rows)
- [ ] Verify row counts match expected
- [ ] Verify products.application_number joins to applications.appl_no
- [ ] Verify search_vector is populated (tsvector trigger fires)
- [ ] Full load with all rows (verify performance < 2 min target)
- [ ] Write E2E test that validates the full pipeline against Docker Compose

### Phase 2: Query API Spec (inline from PRD) (~1 hour)
- [ ] Create specs/query-api.md from PRD section 6
- [ ] Define acceptance criteria for each endpoint
- [ ] Define NDC normalization rules (4-4-2, 5-3-2, 5-4-1 patterns)
- [ ] Define test cases

### Phase 3: NDC Normalization (~3 hours)
- [ ] `internal/api/ndc.go` — NDC format parser and normalizer
- [ ] Accept: hyphenated (0002-1433), unhyphenated (00021433), 3-segment (0002-1433-02)
- [ ] Handle all FDA segment patterns: 4-4-2, 5-3-2, 5-4-1
- [ ] Normalize to canonical hyphenated form for DB lookup
- [ ] Unit tests with all format variants

### Phase 4: Query Endpoints (~6 hours)
- [ ] `GET /api/ndc/{ndc}` — Single NDC lookup with normalization
  - 2-segment: return product + all packages
  - 3-segment: return parent product, highlight matched package
  - Formats: any common format accepted
- [ ] `GET /api/ndc/search?q={query}&limit=50&offset=0` — Full-text search
  - PostgreSQL ts_query with ts_rank ordering
  - Prefix matching (metfor -> metformin)
  - Pagination with total count
- [ ] `GET /api/ndc/{ndc}/packages` — List packages for a product NDC
- [ ] `GET /api/ndc/stats` — Dataset statistics (counts, last load, freshness)
- [ ] All endpoints behind API key middleware
- [ ] Prometheus query/search duration histograms

### Phase 5: Query Store Layer (~3 hours)
- [ ] `internal/store/queries.go` — query functions using pgx
  - LookupByNDC (exact + normalized)
  - SearchProducts (tsvector query)
  - GetPackagesByProductNDC
  - GetStats
- [ ] Interface for query store (for mocking)
- [ ] Unit tests with mocks

### Phase 6: Integration + E2E Tests (~2 hours)
- [ ] Integration tests: query endpoints against real PostgreSQL with test data
- [ ] E2E test: full pipeline (load data -> query -> verify results)
- [ ] Coverage check: target 90%+ for new code
- [ ] golangci-lint clean

## Validation Criteria

### Per-Item Validation
- **fda-data-fetcher E2E**: Real FDA data loaded, row counts verified, joins work
- **query-api**: All 4 endpoints return correct data, NDC normalization handles all formats, search returns ranked results, pagination works

### Cycle Success Criteria
- [ ] Real FDA data loads successfully into PostgreSQL
- [ ] All query endpoints functional and tested
- [ ] NDC normalization handles all 3 format patterns
- [ ] Full-text search returns relevant results
- [ ] 90%+ test coverage on new code
- [ ] golangci-lint clean
- [ ] Docker Compose: full stack boots, loads data, serves queries

## Agent Autonomy & Checkpoints

**Mode:** Autonomous (human away ~3 days)

- Execute phases sequentially
- TDD: tests first, then implementation
- Checkpoint learnings after each phase
- If blocked on FDA download: use fixture data and flag for human review
- If blocked on query design: follow PRD section 6 exactly

## Notes

- PRD section 6 has detailed API contracts — use as the spec source
- NDC format patterns from PRD open question #5: "Store canonical + accept any"
- Subset load (1000 rows) for speed during development; full load for final E2E
- drug-cash integration (M4) is NOT in this cycle
