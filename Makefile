# BiletFlow - Phase 1 database foundation
#
# Requires Docker Desktop (or any Docker Engine) with the compose plugin.

COMPOSE ?= docker compose
PSQL     = $(COMPOSE) exec -T db psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1
DB_USER ?= $(shell grep -E '^POSTGRES_USER=' .env 2>/dev/null | cut -d= -f2)
DB_NAME ?= $(shell grep -E '^POSTGRES_DB=' .env 2>/dev/null | cut -d= -f2)
DB_USER := $(if $(DB_USER),$(DB_USER),biletflow)
DB_NAME := $(if $(DB_NAME),$(DB_NAME),biletflow)

.DEFAULT_GOAL := help
.PHONY: help up down reset wait logs status psql schema seed test tables pgadmin \
	api-build api-run api-test api-test-v api-smoke api-check api-fmt \
	web-install web-dev web-build web-lint web-typecheck web-test web-e2e web-check \
	scan-install scan-dev scan-check verify

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

up: ## Start PostgreSQL (applies the schema on the first start)
	@test -f .env || cp .env.example .env
	$(COMPOSE) up -d
	@$(MAKE) --no-print-directory wait

down: ## Stop the containers (keeps the data volume)
	$(COMPOSE) down

reset: ## Destroy the volume and rebuild the database from db/init
	$(COMPOSE) down -v
	@$(MAKE) --no-print-directory up

wait: ## Block until the database reports healthy
	@printf 'waiting for postgres'
	@for i in $$(seq 1 60); do \
		if [ "$$(docker inspect -f '{{.State.Health.Status}}' biletflow-db 2>/dev/null)" = "healthy" ]; then \
			echo " ready"; exit 0; fi; \
		printf '.'; sleep 1; \
	done; echo " TIMEOUT"; exit 1

logs: ## Follow the database logs
	$(COMPOSE) logs -f db

status: ## Show container status
	$(COMPOSE) ps

psql: ## Open an interactive psql shell
	$(COMPOSE) exec db psql -U $(DB_USER) -d $(DB_NAME)

schema: ## Re-apply db/init by hand (the scripts are idempotent)
	$(PSQL) -f /docker-entrypoint-initdb.d/01_extensions.sql
	$(PSQL) -f /docker-entrypoint-initdb.d/02_schema.sql
	@echo "schema applied"

seed: ## Load the demo dataset (safe to run repeatedly)
	$(PSQL) -q -f /opt/biletflow/seed/01_demo_data.sql

test: ## Run the database test suite
	@./db/tests/run_tests.sh

tables: ## List the tables and their row counts
	@$(PSQL) -c "SELECT relname AS table, n_live_tup AS approx_rows \
		FROM pg_stat_user_tables ORDER BY relname;"

pgadmin: ## Start pgAdmin at http://localhost:5050
	$(COMPOSE) --profile tools up -d pgadmin
	@echo "pgAdmin: http://localhost:5050"

# --- Phase 2: Go API ---------------------------------------------------------

api-build: ## Compile the API to api/bin/api
	cd api && go build -o bin/api ./cmd/api
	@echo "built api/bin/api"

api-run: ## Run the API on http://localhost:8080 (needs `make up`)
	@test -f .env || cp .env.example .env
	@set -a; . ./.env; set +a; cd api && go run ./cmd/api

api-test: ## Run the Go test suite (unit + integration against PostgreSQL)
	cd api && go test ./... -count=1

api-test-v: ## Run the Go test suite verbosely
	cd api && go test ./... -count=1 -v

api-smoke: ## Run the cURL acceptance checks against a running API
	@./api/scripts/smoke_test.sh

api-check: ## Format check, vet and test
	cd api && gofmt -l . && go vet ./... && go test ./... -count=1

api-fmt: ## Format the Go sources
	cd api && gofmt -w .

# --- Phase 3: Next.js organizer dashboard ------------------------------------

web-install: ## Install the frontend dependencies
	cd web && npm install

web-dev: ## Run the dashboard on http://localhost:3000 (needs `make api-run`)
	cd web && npm run dev

web-build: ## Production build of the dashboard
	cd web && npm run build

web-lint: ## Lint the frontend
	cd web && npm run lint

web-typecheck: ## Typecheck the frontend
	cd web && npm run typecheck

web-test: ## Run the frontend unit tests (Vitest)
	cd web && npm run test

web-e2e: ## Run the browser end-to-end suite (needs `make api-run` and `make web-dev`)
	cd web && npm run e2e

web-check: ## Lint, typecheck and unit-test the frontend
	cd web && npm run lint && npm run typecheck && npm run test

# --- Phase 6: Expo ticket scanner --------------------------------------------

scan-install: ## Install the scanner app dependencies
	cd mobile && npm install

scan-dev: ## Run the scanner app (needs `make api-run`)
	cd mobile && npx expo start

scan-check: ## Typecheck the scanner app
	cd mobile && npx tsc --noEmit

# --- Everything ---------------------------------------------------------------

verify: ## Every suite that does not need a browser or a running server
	@$(MAKE) --no-print-directory test
	@$(MAKE) --no-print-directory api-check
	@$(MAKE) --no-print-directory web-check
	@$(MAKE) --no-print-directory scan-check
	@echo ""
	@echo "All offline suites passed. For the rest, with the stack running:"
	@echo "  make api-smoke   # cURL acceptance checks"
	@echo "  make web-e2e     # browser end-to-end suite"
