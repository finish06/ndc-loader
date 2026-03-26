# Cycle 1 — FDA Data Fetcher (Full Implementation)

**Milestone:** M1 — Schema + Loader
**Maturity:** Alpha
**Status:** PLANNED
**Started:** TBD
**Completed:** TBD
**Duration Budget:** 5 days (autonomous — human away for 1 week+)

## Work Items

| Feature | Current Pos | Target Pos | Assigned | Est. Effort | Validation |
|---------|-------------|-----------|----------|-------------|------------|
| fda-data-fetcher | SPECCED | VERIFIED | Agent-1 | ~20 hours | All 18 acceptance criteria passing in tests, 90%+ coverage |

## Implementation Order (Serial — Alpha Maturity)

Single-threaded execution. Features advance sequentially through these phases:

### Phase 1: Project Scaffold (~2 hours)
- [ ] `go mod init` with module path
- [ ] Directory structure per CLAUDE.md: `cmd/`, `internal/{api,loader,store,model}/`, `migrations/`
- [ ] `docker-compose.yml` with PostgreSQL 16 + ndc-loader service
- [ ] `Dockerfile` (multi-stage build, scratch/alpine final)
- [ ] `datasets.yaml` configuration file
- [ ] Environment variable loading
- [ ] Structured JSON logger setup
- [ ] Basic `cmd/ndc-loader/main.go` entrypoint

### Phase 2: Database Schema + Migrations (~3 hours)
- [ ] SQL migration files for all 9 tables (products, packages, applications, drugsfda_products, submissions, marketing_status, active_ingredients, te_codes, load_checkpoints)
- [ ] All indexes (12 defined in spec)
- [ ] Migration runner (embed SQL, run on startup)
- [ ] `internal/store/` — database connection pool (pgx)
- [ ] Unit tests for migration runner

### Phase 3: Download + Discovery (~4 hours)
- [ ] `internal/loader/discovery.go` — fetch openFDA downloads page, parse available datasets
- [ ] `internal/loader/downloader.go` — download ZIP files with retry (exponential backoff, max 3 attempts)
- [ ] ZIP extraction to temp directory
- [ ] Integrity checks on downloaded ZIPs
- [ ] Dataset configuration loading from `datasets.yaml`
- [ ] Unit tests for discovery, downloader, extraction
- [ ] Integration test: download real ndctext.zip (small test fixture for unit tests)

### Phase 4: Parser + Data Loading (~5 hours)
- [ ] `internal/loader/parser.go` — tab-delimited file parser (generic, handles any dataset)
- [ ] `internal/store/loader.go` — bulk COPY FROM for each table
- [ ] Atomic swap per table (staging table → live table rename within transaction)
- [ ] Row count validation (>20% drop safety valve)
- [ ] Column mapping: handle unexpected/missing columns gracefully
- [ ] Skip malformed rows with warning log
- [ ] Full-text search vector generation for products table
- [ ] Unit tests for parser (with fixture data)
- [ ] Integration tests for bulk load + atomic swap

### Phase 5: Checkpoint System (~3 hours)
- [ ] `internal/loader/checkpoint.go` — checkpoint manager
- [ ] Track per-table status: pending → downloading → downloaded → loading → loaded → failed
- [ ] Load ID (UUID) per execution
- [ ] Resume logic: skip tables with status=loaded, retry tables with status=failed
- [ ] Record timing, row counts, error messages
- [ ] Unit tests for checkpoint state machine
- [ ] Integration test: simulate failure at table N, verify resume at N

### Phase 6: API Key Authentication + Admin Endpoints (~3 hours)
- [ ] `internal/api/middleware.go` — X-API-Key authentication middleware
- [ ] API key loading from `API_KEYS` env var (comma-separated)
- [ ] 401 response for missing/invalid keys
- [ ] `internal/api/router.go` — Chi router setup
- [ ] `POST /api/admin/load` — trigger manual load (202 Accepted, 409 if in progress)
- [ ] `GET /api/admin/load/{load_id}` — check load status with checkpoint details
- [ ] Unit tests for auth middleware
- [ ] Integration tests for admin endpoints

### Phase 7: Scheduler + Wiring (~2 hours)
- [ ] Cron-based scheduler (`robfig/cron` or equivalent)
- [ ] Wire all components together in `main.go`
- [ ] Graceful shutdown (context cancellation)
- [ ] Docker Compose: verify full stack boots and loads data
- [ ] E2E test: full lifecycle (start → download → parse → load → query admin endpoint)

### Phase 8: Hardening + Quality Gates (~3 hours)
- [ ] `golangci-lint` configuration
- [ ] Coverage check (target: 90%)
- [ ] All 11 edge cases from spec tested
- [ ] Prometheus metrics integration (load duration, row counts, errors, query latency)
- [ ] Health endpoint with data freshness
- [ ] Final `go vet` pass
- [ ] Documentation: README.md with setup instructions

## Dependencies & Serialization

```
Phase 1 (scaffold)
    ↓
Phase 2 (schema)
    ↓
Phase 3 (download) ─── Phase 4 (parser) depends on Phase 3 fixtures
    ↓                      ↓
Phase 5 (checkpoints) ← uses store + loader
    ↓
Phase 6 (API + auth)
    ↓
Phase 7 (scheduler + wiring)
    ↓
Phase 8 (hardening)
```

Single-threaded execution. All phases are serial.

## Validation Criteria

### Per-Phase Validation
- **Phase 1:** `docker-compose up` boots PostgreSQL, `go build ./...` succeeds
- **Phase 2:** Migrations run, all 9 tables created, indexes verified
- **Phase 3:** ZIP downloaded, extracted, files accessible. Unit tests pass.
- **Phase 4:** Products + packages loaded from fixture data. Atomic swap verified. Row count check works.
- **Phase 5:** Checkpoint state machine tested. Resume-from-failure verified.
- **Phase 6:** API key auth blocks unauthorized requests. Admin endpoints return correct load status.
- **Phase 7:** Full lifecycle works end-to-end in Docker Compose.
- **Phase 8:** 90%+ coverage, all lints pass, all edge cases tested.

### Cycle Success Criteria
- [ ] All 18 acceptance criteria from spec verified in tests
- [ ] 90%+ test coverage (strict quality mode)
- [ ] `golangci-lint run` passes clean
- [ ] `go vet ./...` passes clean
- [ ] Docker Compose boots and loads data successfully
- [ ] All 11 edge cases from spec have test coverage
- [ ] Checkpoint recovery works: partial failure → resume at failure point
- [ ] API key authentication blocks all unauthenticated requests

## Agent Autonomy & Checkpoints

**Mode:** Full autonomous (human away for 1 week+)

- Agent executes all phases sequentially
- TDD approach: write tests first, then implementation
- Checkpoint `.add/learnings.md` after each phase completion
- If blocked: log blocker in learnings, attempt workaround, continue with next phase if possible
- Final comprehensive checkpoint when cycle completes
- No human approval needed until cycle review

## Test Strategy

**TDD flow per phase:**
1. Write failing tests for the phase's acceptance criteria
2. Implement minimum code to pass tests
3. Refactor for clarity
4. Verify coverage meets 90% threshold
5. Move to next phase

**Test fixtures:**
- Create small fixture files (10-20 rows) mimicking FDA tab-delimited format
- Use these for unit tests (fast, no network)
- Integration tests use real PostgreSQL via Docker Compose
- E2E test does a full download-parse-load cycle (can use fixture or real data)

## Notes

- Drugs@FDA actual download URL and file structure should be verified during Phase 3 — the spec uses placeholder URLs that need confirmation against the real openFDA downloads page
- The openFDA downloads page may require parsing JSON API responses rather than HTML scraping — discovery implementation should handle both
- `application_number` join key format may need normalization (NDC uses "NDA012345", Drugs@FDA may use just "012345")
