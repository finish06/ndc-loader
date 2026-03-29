# Cycle 4 — Swagger + Health/Version + Pharm Classes + Staging Deploy

**Milestone:** M3 — Ops & Polish
**Maturity:** Alpha
**Status:** IN_PROGRESS
**Started:** 2026-03-27
**Completed:** TBD
**Duration Budget:** 2 days

## Work Items

| Feature | Current Pos | Target Pos | Status | Notes |
|---------|-------------|-----------|--------|-------|
| swagger-docs | SPECCED | VERIFIED | DONE | Swagger UI at /swagger/, all handlers annotated |
| pharm-classes-structured | SHAPED | VERIFIED | DONE | ParsePharmClasses + enrichProduct on internal API |
| health-version-endpoints | SPECCED | IN_PROGRESS | WIP | version.go + health.go written, main.go needs wiring |
| staging-deploy | SHAPED | VERIFIED | DONE | CI pipeline, docker-compose.staging, Makefile, deploy hook |
| ci-pipeline | SHAPED | VERIFIED | DONE | GitHub Actions: lint, test, publish, staging webhook |

## Remaining Work

1. Wire health/version into main.go (ldflags + pass db to router)
2. Remove old healthHandler from handlers.go
3. Update Dockerfile + Makefile with ldflags
4. Tests for health + version handlers
5. Commit + push + verify CI passes
