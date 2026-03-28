# Cycle 4 — Swagger API Documentation

**Milestone:** M2 (addendum — ops enhancement)
**Maturity:** Alpha
**Status:** PLANNED
**Started:** TBD
**Completed:** TBD
**Duration Budget:** 1 day

## Work Items

| Feature | Current Pos | Target Pos | Assigned | Est. Effort | Validation |
|---------|-------------|-----------|----------|-------------|------------|
| swagger-docs | SPECCED | VERIFIED | Agent-1 | ~4 hours | All 12 AC passing, Swagger UI serves at /swagger/ |

## Dependencies & Serialization

No dependencies. Single feature, single-threaded.

## Implementation Order

### Phase 1: Install swaggo dependencies (~15 min)
- [ ] `go install github.com/swaggo/swag/v2/cmd/swag@latest`
- [ ] `go get github.com/swaggo/http-swagger/v2`
- [ ] `go get github.com/swaggo/swag/v2`
- [ ] Verify `swag init` runs without error

### Phase 2: Annotate main.go (~15 min)
- [ ] Add top-level `@title`, `@version`, `@description`, `@host`, `@BasePath`
- [ ] Add `@securityDefinitions.apikey ApiKeyAuth` with `@in header` `@name X-API-Key`
- [ ] Run `swag init -g cmd/ndc-loader/main.go` to generate docs package

### Phase 3: Annotate all handlers (~2 hours)
- [ ] Query handlers: LookupNDC, SearchNDC, ListPackages, GetStats
  - `@Summary`, `@Description`, `@Tags Query`
  - `@Param` for path/query params + API key header
  - `@Success 200` with response type reference
  - `@Failure 400,401,404` with error type
  - `@Router` with method
- [ ] Admin handlers: TriggerLoad, GetLoadStatus
  - `@Tags Admin`
  - Request body schema for TriggerLoad
- [ ] openFDA handler: HandleNDCJSON
  - `@Tags OpenFDA`
  - Document search/limit/skip params
  - Reference OpenFDAResponse type
- [ ] Health handler
  - `@Tags Operations`
  - No auth required notation
- [ ] Response type annotations on structs (ProductResult, SearchResult, etc.)

### Phase 4: Register Swagger UI route (~15 min)
- [ ] Import `httpSwagger` and generated `docs` package
- [ ] Register `r.Get("/swagger/*", httpSwagger.WrapHandler)` (no auth)
- [ ] Verify UI loads at http://localhost:8081/swagger/

### Phase 5: Makefile / go generate (~15 min)
- [ ] Add `//go:generate swag init -g cmd/ndc-loader/main.go` directive
- [ ] Or add Makefile target: `make docs`
- [ ] Commit generated `docs/` package

### Phase 6: Tests + Quality (~1 hour)
- [ ] Unit test: GET /swagger/index.html returns 200
- [ ] Verify `swag init` produces valid JSON (parse with encoding/json)
- [ ] golangci-lint clean
- [ ] All existing tests still pass

## Validation Criteria

### Cycle Success Criteria
- [ ] Swagger UI loads at /swagger/
- [ ] All endpoints documented with params, responses, examples
- [ ] API key auth scheme documented
- [ ] Error responses (400, 401, 404, 500) documented
- [ ] `swag init` regenerates cleanly
- [ ] golangci-lint clean
- [ ] No regressions

## Agent Autonomy & Checkpoints

**Mode:** Per user availability — ask before starting.

## Notes

- swaggo/swag v2 supports OpenAPI 3.0 (v1 only does Swagger 2.0)
- Generated docs/ package should be committed so Docker builds don't need swag CLI
- Swagger UI is served without auth (internal service)
