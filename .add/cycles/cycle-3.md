# Cycle 3 — openFDA-Compatible API for drug-cash

**Milestone:** M2 — drug-cash Integration
**Maturity:** Alpha
**Status:** PLANNED
**Started:** TBD
**Completed:** TBD
**Duration Budget:** 2 days (autonomous — human away 1-2 days)

## Work Items

| Feature | Current Pos | Target Pos | Assigned | Est. Effort | Validation |
|---------|-------------|-----------|----------|-------------|------------|
| openfda-compat-api | SHAPED | VERIFIED | Agent-1 | ~10 hours | All 14 AC passing, openFDA format parity verified |

## Dependencies & Serialization

No dependencies — single feature, builds on existing query store.

Single-threaded execution.

## Implementation Order

### Phase 1: openFDA Response Types (~1 hour)
- [ ] `internal/api/openfda_types.go` — response structs matching openFDA JSON schema
  - OpenFDAResponse (meta + results)
  - OpenFDAMeta (disclaimer, terms, license, last_updated, results pagination)
  - OpenFDAProduct (all fields: product_ndc, brand_name, generic_name, active_ingredients array, packaging array, openfda nested object, route array, pharm_class array)
- [ ] Unit tests for struct serialization (marshal to JSON, verify field names)

### Phase 2: Field Mapping / Transform (~2 hours)
- [ ] `internal/api/openfda_transform.go` — transform DB rows to openFDA format
  - Split substance_name (semicolon-delimited) into active_ingredients [{name, strength}] array
  - Split route (semicolon-delimited) into string array
  - Split pharm_classes (semicolon-delimited) into pharm_class array
  - Wrap labeler_name into openfda.manufacturer_name array
  - Format marketing_start as YYYYMMDD string (not time.Time)
  - Map packages to packaging array format
  - product_type, marketing_category, application_number as-is
  - spl_id, finished, brand_name_base defaults
- [ ] Unit tests for each transform function with real FDA data samples

### Phase 3: Search Parameter Parser (~2 hours)
- [ ] `internal/api/openfda_search.go` — parse openFDA search syntax
  - `brand_name:metformin` → field-specific query
  - `brand_name:"Metformin Hydrochloride"` → exact phrase
  - `metformin` → full-text search (no field prefix)
  - `brand_name:metformin+generic_name:hydrochloride` → AND combination
  - Map openFDA field names to DB columns
- [ ] Unit tests for all search syntax variants

### Phase 4: Handler + Route (~2 hours)
- [ ] `internal/api/openfda_handler.go` — GET /api/openfda/ndc.json handler
  - Parse search, limit (default 1, max 1000), skip parameters
  - Route to appropriate query store method
  - Build openFDA-format response with meta
  - 404 with openFDA error format when no results
- [ ] Register route in router.go (behind API key auth)
- [ ] Integration test against real loaded FDA data

### Phase 5: Format Parity Validation (~2 hours)
- [ ] E2E test: fetch same query from real openFDA API and ndc-loader, compare response structure
  - Verify all field names match
  - Verify array types match (active_ingredients, packaging, route, pharm_class)
  - Verify meta pagination format matches
  - Verify 404 error format matches
- [ ] Test with drug-cash slug config pattern (search param format)

### Phase 6: Quality Gates (~1 hour)
- [ ] 90%+ coverage on new code
- [ ] golangci-lint clean
- [ ] go vet clean
- [ ] All existing tests still pass (no regressions)

## Validation Criteria

### Per-Item Validation
- **openfda-compat-api**: All 14 AC from spec verified in tests

### Cycle Success Criteria
- [ ] GET /api/openfda/ndc.json returns openFDA-format response
- [ ] Response structure matches real openFDA API
- [ ] search parameter supports field:value syntax
- [ ] Pagination via skip/limit matches openFDA behavior
- [ ] 404 error matches openFDA format
- [ ] active_ingredients is array of {name, strength}
- [ ] packaging is array with correct fields
- [ ] route and pharm_class are string arrays
- [ ] 90%+ test coverage on new code
- [ ] golangci-lint clean

## Agent Autonomy & Checkpoints

**Mode:** Autonomous (human away 1-2 days)

- Execute phases sequentially
- TDD: tests first where practical
- Checkpoint learnings after completion
- Validate format parity against real openFDA API
- If openFDA API is down: use snapshot fixture and flag for human

## Notes

- drug-cash slug config lives in the drug-cash repo, not here
- openFDA defaults limit=1 (not 50 like our internal API) — match their default
- openFDA active_ingredients splits substance_name + strength into separate objects
- Some openFDA fields we don't have (rxcui, spl_set_id, upc, unii) — return empty arrays
