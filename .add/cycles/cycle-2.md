# Cycle 2 — E2E FDA Validation + Query API

**Milestone:** M1 (close) + M2 (start)
**Maturity:** Alpha
**Status:** COMPLETE
**Started:** 2026-03-26
**Completed:** 2026-03-26
**Duration Budget:** 3 days (used ~1 day)

## Work Items

| Feature | Current Pos | Target Pos | Assigned | Est. Effort | Validation |
|---------|-------------|-----------|----------|-------------|------------|
| fda-data-fetcher (E2E) | IN_PROGRESS | VERIFIED | Agent-1 | ~3 hours | Real FDA data loaded, joins verified, search works |
| query-api | SHAPED | VERIFIED | Agent-1 | ~8 hours | 4 endpoints, NDC normalization, 11 E2E tests passing |

## Results

- Real FDA data: 112K products, 212K packages, 29K applications, 51K drugsfda products, 191K submissions
- Join validation: 59K products match applications via normalized application_number
- Full-text search: 586 results for "metformin", prefix matching works
- 4 query endpoints: lookup, search, packages, stats
- NDC normalization: 4-4-2, 5-3-2, 5-4-1 segment patterns
- 11 E2E query tests passing against real FDA data
- Load time: ~6.5 seconds (target <2 minutes)

## Bugs Found & Fixed

1. FDA data contains Windows-1252 bytes → UTF-8 sanitization added
2. FK constraints fail with real FDA data (orphaned references) → removed FKs
3. application_number format mismatch (ANDA076543 vs 076543) → normalized join
4. Real FDA schema differs from spec (PROPRIETARYNAMESUFFIX, SponsorName, etc.) → updated schema + mappings
5. datasets.yaml file list wrong (ActionTypes_Lookup is not active ingredients) → corrected

## Learnings

- FDA data quality is inconsistent: orphaned FKs between datasets, non-UTF8 encoding, format mismatches
- Real data validation is essential — spec assumptions were wrong on multiple fronts
- Removing FK constraints was the right call for cross-system bulk data
- application_number join requires normalization (strip alpha prefix, zero-pad to 6 digits)
