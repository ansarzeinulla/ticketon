# BiletFlow - local development
#
# Requires Docker with the compose plugin.

COMPOSE ?= docker compose
DB_USER ?= biletflow
DB_NAME ?= biletflow

.DEFAULT_GOAL := help
.PHONY: help up down reset wait psql test api-run api-test api-check \
	web-install web-dev web-check scan-install scan-dev scan-check

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

psql: ## Open an interactive psql shell
	$(COMPOSE) exec db psql -U $(DB_USER) -d $(DB_NAME)

test: ## Run the database test suite (needs `make up`)
	@./db/tests/run_tests.sh

api-run: ## Run the API on http://localhost:8080 (needs `make up`)
	@test -f .env || cp .env.example .env
	@set -a; . ./.env; set +a; cd api && go run ./cmd/api

api-test: ## Run the Go tests (integration tests need `make up`)
	cd api && go test ./... -count=1

api-check: ## Go: format check, vet and tests
	@cd api && test -z "$$(gofmt -l .)" || (echo "run gofmt on:"; gofmt -l .; exit 1)
	cd api && go vet ./...
	cd api && go test ./...

web-install: ## Install the web dependencies
	cd web && npm install

web-dev: ## Run the web app on http://localhost:3000 (needs `make api-run`)
	cd web && npm run dev

web-check: ## Web: lint, typecheck and production build
	cd web && npm run lint
	cd web && npm run typecheck
	cd web && npm run build

scan-install: ## Install the mobile app dependencies
	cd mobile && npm install

scan-dev: ## Run the mobile app with Expo (needs `make api-run`)
	cd mobile && npx expo start

scan-check: ## Typecheck the mobile app
	cd mobile && npx tsc --noEmit
