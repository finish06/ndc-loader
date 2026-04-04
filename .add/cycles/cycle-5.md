# Cycle 5 — Landing Page + Release Tagging

**Milestone:** M4 — Production Readiness
**Maturity:** Beta
**Status:** COMPLETE
**Started:** 2026-04-03
**Completed:** 2026-04-04

## Work Items

| Feature | Current Pos | Target Pos | Status | Notes |
|---------|-------------|-----------|--------|-------|
| landing-page | SPECCED | VERIFIED | DONE | landing/index.html + redirect handler + GitHub Pages |
| release-tagging | SHAPED | VERIFIED | DONE | Spec created, `make release` + CI GitHub Release implemented |

## Results

- Landing page deployed to GitHub Pages (rx-dag.calebdunn.tech)
- Root redirect handler (GET / → LANDING_URL)
- Release tagging spec (8 AC)
- `make release VERSION=x.y.z` convenience target
- CI creates GitHub Release on v* tags
- v0.2.0 tagged and released
- Fixed CI --generate-notes + --notes conflict (PR #4)
- 4 PRs merged via branch protection workflow
