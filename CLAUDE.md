# ndc-loader

FDA NDC Directory bulk loader and REST API service. Ingests the complete NDC Directory daily from FDA bulk download and serves it via REST API, replacing openFDA API dependency for drug-cash and internal microservices.

## Methodology

This project follows **Agent Driven Development (ADD)** — specs drive agents, humans architect and decide, trust-but-verify ensures quality.

- **PRD:** docs/prd.md
- **Specs:** specs/
- **Plans:** docs/plans/
- **Config:** .add/config.json

Document hierarchy: PRD -> Spec -> Plan -> User Test Cases -> Automated Tests -> Implementation

## Tech Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Language | Go | 1.22+ |
| HTTP Framework | Chi | v5 |
| Database | PostgreSQL | 16+ |
| DB Driver | pgx | v5 |
| Containers | Docker Compose | — |
| Metrics | Prometheus | — |

## Commands

### Development
```
docker-compose up                    # Start local dev (PostgreSQL + ndc-loader)
go test -race -cover ./...           # Run unit tests
go test -tags=integration ./tests/integration/...  # Integration tests
go test -tags=e2e ./tests/e2e/...    # E2E tests
golangci-lint run                    # Lint check
go vet ./...                         # Vet check
```

### ADD Workflow
```
/add:spec {feature}                  # Create feature specification
/add:plan specs/{feature}.md         # Create implementation plan
/add:tdd-cycle specs/{feature}.md    # Execute TDD cycle
/add:verify                          # Run quality gates
/add:deploy                          # Commit and deploy
/add:away {duration}                 # Human stepping away
/add:retro                           # Run retrospective
```

## Architecture

### Key Directories
```
ndc-loader/
  cmd/                  # Application entrypoints
  internal/             # Private application code
    api/                # HTTP handlers and routing
    loader/             # FDA data download and parsing
    store/              # Database access layer
    model/              # Domain types
  migrations/           # SQL migration files
  tests/
    unit/               # Unit tests
    integration/        # Integration tests (requires DB)
    e2e/                # End-to-end API tests
    screenshots/        # Visual verification artifacts
  specs/                # Feature specifications
  docs/
    prd.md              # Product Requirements Document
    plans/              # Implementation plans
    milestones/         # Milestone tracking
  .add/                 # ADD methodology config and learnings
  docker-compose.yml    # Local dev environment
  Dockerfile            # Production container
```

### Environments

- **Local:** docker-compose up — PostgreSQL + ndc-loader on http://localhost:8081
- **Staging:** Push to staging branch — pre-production validation
- **Production:** Merge to main — self-hosted homelab (dockerhub.calebdunn.tech)

## Quality Gates

- **Mode:** Strict
- **Coverage threshold:** 90%
- **Type checking:** Blocking (go vet)
- **E2E required:** Yes

All gates defined in `.add/config.json`. Run `/add:verify` to check.

## Source Control

- **Git host:** GitHub
- **Branching:** Feature branches off `main`
- **Commits:** Conventional commits (feat:, fix:, test:, refactor:, docs:)
- **CI/CD:** GitHub Actions

## Collaboration

- **Autonomy level:** Autonomous
- **Review gates:** PR review before merge
- **Deploy approval:** Required for production

## ADD Methodology

**Configuration:** `.add/config.json`
**Knowledge Base:** `.add/learnings.md` (auto-populated during development)
**Quality Gates:** Enforced via `/add:verify` command
**Workflow:** `/add:spec` -> `/add:plan` -> `/add:tdd-cycle` -> `/add:retro`

See `.add/config.json` for detailed settings.
