# Active Learnings (1 of 1)

> Pre-filtered by severity and date. Full data: `.add/learnings.json`

### process
- **[medium]** Swagger spec drifts silently — regenerate on every /add:docs run (L-001, 2026-05-16)
  docs/swagger/{swagger.json,yaml,docs.go} are committed but not rebuilt by CI. Between Apr 4 and May 16, the openFDA handler, root redirect, and several response shapes shipped without a spec regen — Swagger UI was lying. Fix: `make docs` (or `go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/ndc-loader/main.go -o docs/swagger`) is cheap; run it as part of any docs/api change and consider wiring it into pre-push or CI.

---
*Auto-generated. Do not edit — regenerated on each learning write.*
