# Milestone M4 — Production Readiness

**Goal:** Stable release with production deployment, monitoring, and public rx-dag landing page.
**Status:** IN PROGRESS
**Appetite:** 1 week
**Target Maturity:** Beta

## Hill Chart

```
release-tagging       ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  SHAPED
deploy-workflow       ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  SHAPED
sla-monitoring        ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  SHAPED
perf-baselines        ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  SHAPED
landing-page          ██████████████░░░░░░░░░░░░░░░░░░░░░░░  SPECCED
```

## Features

| Feature | Spec | Position | Target | Notes |
|---------|------|----------|--------|-------|
| landing-page | specs/landing-page.md | SPECCED | VERIFIED | rx-dag public landing page + LANDING_URL root redirect |
| release-tagging | — | SHAPED | VERIFIED | Semantic versioning, tagged releases |
| deploy-workflow | — | SHAPED | VERIFIED | Production deployment via tagged release |
| sla-monitoring | — | SHAPED | VERIFIED | Uptime monitoring and alerting |
| perf-baselines | — | SHAPED | VERIFIED | Response time baselines established |

## Success Criteria

- [ ] Tagged v1.0.0 release
- [ ] Production deploy via tagged release
- [ ] Response time baselines established
- [ ] Uptime monitoring configured
- [ ] rx-dag landing page live on GitHub Pages
- [ ] GET / redirects to landing page via LANDING_URL env var
