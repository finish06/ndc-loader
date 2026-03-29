# Deployment Guide

## Environments

| Environment | Host | Port | Deploy Trigger |
|-------------|------|------|----------------|
| Local | localhost | 8081 | `docker compose up` |
| Staging | 192.168.1.145 | 8081 | Push to main (auto) or `make deploy-staging` |
| Production | TBD | 8081 | Tag `v*` (manual merge to main) |

## Local Development

```bash
cp .env.example .env        # Create env file
docker compose up -d         # Start postgres + ndc-loader
curl http://localhost:8081/health  # Verify
```

Trigger initial data load:
```bash
curl -X POST http://localhost:8081/api/admin/load \
  -H "X-API-Key: dev-key-change-me" \
  -H "Content-Type: application/json" -d '{}'
```

## Staging Deployment

### First Time Setup

```bash
make deploy-staging-first
# Then SSH to staging and edit /opt/rx-dag/.env with real credentials
ssh finish06@192.168.1.145 "vi /opt/rx-dag/.env"
make staging-start
make staging-load  # Trigger initial FDA data load
```

### Routine Deploy

Automatic on push to main:
1. GitHub Actions builds `rx-dag:beta` image
2. Pushes to `dockerhub.calebdunn.tech/finish06/rx-dag:beta`
3. Triggers staging deploy webhook
4. Deploy hook on staging1 pulls + restarts container
5. Health check verifies service is up

Manual:
```bash
make deploy-staging  # Sync config + pull + restart + health check
```

### Staging Operations

```bash
make staging-status   # Health check + container status
make staging-logs     # Tail container logs
make staging-restart  # Pull latest + restart (keeps postgres data)
make staging-load     # Trigger FDA data refresh
make staging-psql     # Open psql shell on staging DB
```

### pgAdmin (Optional)

Start the database UI on staging:
```bash
ssh finish06@192.168.1.145 "cd /opt/rx-dag && docker compose --profile debug up -d pgadmin"
```
Access at `http://192.168.1.145:5050`. Connect to `ndc-loader-postgres:5432`.

## CI/CD Pipeline

### Triggers

| Event | Jobs | Image Tag |
|-------|------|-----------|
| Push to `main` | lint + test + publish + staging deploy | `:beta` |
| PR to `main` | lint + test (no publish) | — |
| Tag `v*` | lint + test + publish | `:version` + `:latest` |

### Jobs

1. **Lint** — `golangci-lint run ./...`
2. **Test** — unit tests + integration tests (PostgreSQL 16 service container)
3. **Vet** — `go vet ./...`
4. **Publish** (main/tags only):
   - Build Docker image with ldflags (version, git commit, branch)
   - Push to `dockerhub.calebdunn.tech/finish06/rx-dag`
   - Push to `ghcr.io/finish06/rx-dag`
   - Trigger staging deploy webhook

### Required GitHub Secrets

| Secret | Description |
|--------|-------------|
| `REGISTRY_USERNAME` | dockerhub.calebdunn.tech login |
| `REGISTRY_PASSWORD` | dockerhub.calebdunn.tech password |
| `WEBHOOK_SECRET` | HMAC key for deploy webhook signature |
| `STAGING_WEBHOOK_URL` | Deploy hook endpoint (https://deploy.staging1.calebdunn.tech) |

### Deploy Hook Configuration

On staging1, the deploy hook config needs:
```yaml
rx-dag:
  compose_dir: /opt/rx-dag
  compose_file: docker-compose.yml
  health_checks:
    - name: health
      url: http://192.168.1.145:8081/health
      expect_key: status
```

## Docker Compose Files

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Local development |
| `docker-compose.staging.yml` | Staging (copied to staging1 as `docker-compose.yml`) |

## Release Process

1. Ensure all tests pass: `make test && make lint`
2. Tag the release: `git tag v0.1.0 && git push --tags`
3. CI builds and pushes `:0.1.0` + `:latest` to both registries
4. Deploy to production (manual process TBD)

## Environment Variables

See `.env.example` for the full list. Key variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `API_KEYS` | yes | Comma-separated API keys for authentication |
| `DATABASE_URL` | yes | PostgreSQL connection string |
| `LOAD_SCHEDULE` | no | Cron expression for daily FDA refresh (default: `0 3 * * *`) |
| `LOG_LEVEL` | no | `debug`, `info`, `warn`, `error` (default: `info`) |
