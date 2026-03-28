# ndc-loader Makefile

STAGING_HOST ?= 192.168.1.145
STAGING_USER ?= $(USER)
STAGING_DIR  ?= /opt/rx-dag
REGISTRY     ?= dockerhub.calebdunn.tech/finish06/ndc-loader

.PHONY: build test lint docs deploy-staging staging-logs staging-status staging-restart

# ── Development ──────────────────────────────────────────────

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ndc-loader ./cmd/ndc-loader

test:
	go test -race -cover ./internal/...

test-integration:
	DATABASE_URL="postgres://ndc:ndc@localhost:5435/ndc?sslmode=disable" \
		go test -tags=integration -race ./tests/integration/...

test-e2e:
	DATABASE_URL="postgres://ndc:ndc@localhost:5435/ndc?sslmode=disable" \
		go test -tags=e2e -v -timeout=10m ./tests/e2e/...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

docs:
	swag init -g cmd/ndc-loader/main.go -o docs/swagger

# ── Docker ───────────────────────────────────────────────────

docker-build:
	docker build -t $(REGISTRY):local .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

# ── Staging Deploy (192.168.1.145) ───────────────────────────

deploy-staging: ## Deploy to staging1
	@echo "==> Deploying ndc-loader to staging ($(STAGING_HOST))..."
	@echo "1. Syncing config files..."
	ssh $(STAGING_USER)@$(STAGING_HOST) "mkdir -p $(STAGING_DIR)"
	scp docker-compose.staging.yml $(STAGING_USER)@$(STAGING_HOST):$(STAGING_DIR)/docker-compose.yml
	scp datasets.yaml $(STAGING_USER)@$(STAGING_HOST):$(STAGING_DIR)/datasets.yaml
	@echo "2. Syncing .env (if not exists on remote)..."
	ssh $(STAGING_USER)@$(STAGING_HOST) "test -f $(STAGING_DIR)/.env || echo 'NEEDS_ENV=true'"
	@echo "   NOTE: If first deploy, copy .env.staging to staging and edit passwords:"
	@echo "   scp .env.staging $(STAGING_USER)@$(STAGING_HOST):$(STAGING_DIR)/.env"
	@echo "3. Pulling latest image and restarting..."
	ssh $(STAGING_USER)@$(STAGING_HOST) "\
		cd $(STAGING_DIR) && \
		docker pull $(REGISTRY):beta && \
		docker compose pull && \
		docker compose up -d --force-recreate ndc-loader"
	@echo "4. Waiting for health check..."
	@sleep 5
	@ssh $(STAGING_USER)@$(STAGING_HOST) "curl -sf http://localhost:8081/health" && \
		echo "==> Staging deploy OK" || \
		echo "==> WARNING: Health check failed"

deploy-staging-first: ## First-time staging setup (creates .env, starts everything)
	@echo "==> First-time staging setup on $(STAGING_HOST)..."
	ssh $(STAGING_USER)@$(STAGING_HOST) "mkdir -p $(STAGING_DIR)"
	scp docker-compose.staging.yml $(STAGING_USER)@$(STAGING_HOST):$(STAGING_DIR)/docker-compose.yml
	scp datasets.yaml $(STAGING_USER)@$(STAGING_HOST):$(STAGING_DIR)/datasets.yaml
	scp .env.staging $(STAGING_USER)@$(STAGING_HOST):$(STAGING_DIR)/.env
	@echo "IMPORTANT: SSH into staging and edit $(STAGING_DIR)/.env with real passwords!"
	@echo "Then run: make staging-start"

staging-start: ## Start all services on staging
	ssh $(STAGING_USER)@$(STAGING_HOST) "\
		cd $(STAGING_DIR) && \
		docker compose pull && \
		docker compose up -d"

staging-stop: ## Stop all services on staging
	ssh $(STAGING_USER)@$(STAGING_HOST) "\
		cd $(STAGING_DIR) && \
		docker compose down"

staging-restart: ## Restart ndc-loader on staging (keeps postgres)
	ssh $(STAGING_USER)@$(STAGING_HOST) "\
		cd $(STAGING_DIR) && \
		docker compose pull ndc-loader && \
		docker compose up -d --force-recreate ndc-loader"

staging-logs: ## Tail staging logs
	ssh $(STAGING_USER)@$(STAGING_HOST) "\
		cd $(STAGING_DIR) && \
		docker compose logs -f --tail=50 ndc-loader"

staging-status: ## Check staging health
	@echo "==> Staging status ($(STAGING_HOST)):"
	@ssh $(STAGING_USER)@$(STAGING_HOST) "\
		cd $(STAGING_DIR) && \
		docker compose ps && \
		echo '---' && \
		curl -sf http://localhost:8081/health | python3 -m json.tool"

staging-load: ## Trigger data load on staging
	@echo "==> Triggering FDA data load on staging..."
	ssh $(STAGING_USER)@$(STAGING_HOST) "\
		curl -X POST http://localhost:8081/api/admin/load \
			-H 'Content-Type: application/json' \
			-H 'X-API-Key: \$$(grep API_KEYS $(STAGING_DIR)/.env | cut -d= -f2)' \
			-d '{}'"

staging-psql: ## Open psql on staging postgres
	ssh -t $(STAGING_USER)@$(STAGING_HOST) "\
		cd $(STAGING_DIR) && \
		docker compose exec ndc-loader-postgres psql -U ndc -d ndc"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
