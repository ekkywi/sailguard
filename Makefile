.PHONY: help \
	dev-up dev-down \
	migrate-status migrate-up migrate-down \
	test-pkgs tidy \
	build-api build-worker build-agent build-agent-win7 build-agent-windows \
	build-agent-setup \
	web-install web-dev \
	run-api run-web run-agent run-worker

# Prefer goose on PATH; fall back to $(go env GOPATH)/bin/goose
GOOSE ?= $(shell command -v goose 2>/dev/null || echo $$(go env GOPATH)/bin/goose)
SG_DB_URL ?= postgres://sailguard:sailguard@127.0.0.1:15433/sailguard?sslmode=disable
MIGRATE_DIR := services/controlplane/migrations

# Optional overrides for run-agent (env wins if already set / from .env)
SG_DATA_DIR ?= $(HOME)/.sailguard-agent-lab
SG_SERVER_URL ?= http://127.0.0.1:18080

help:
	@echo "SailGuard targets:"
	@echo ""
	@echo "  Dependencies (Docker):"
	@echo "  make dev-up          Start Postgres + Redis (dev)"
	@echo "  make dev-down        Stop dev dependencies"
	@echo ""
	@echo "  Database:"
	@echo "  make migrate-status  Goose migration status"
	@echo "  make migrate-up      Apply pending migrations"
	@echo "  make migrate-down    Roll back one migration"
	@echo ""
	@echo "  Build (compile binaries to bin/):"
	@echo "  make build-api       Build control plane API"
	@echo "  make build-worker    Build worker"
	@echo "  make build-agent     Build agent for current GOOS"
	@echo "  make build-agent-win7     Win7 SP1+ exe (Go 1.20 script)"
	@echo "  make build-agent-windows  Win10/11 exe (current Go)"
	@echo "  make build-agent-setup    Windows installer exe (Go 1.20)"
	@echo ""
	@echo "  Run (foreground — one terminal each; loads .env if present):"
	@echo "  make run-api         go run API (host)"
	@echo "  make run-web         Vite dev server (same as web-dev)"
	@echo "  make run-agent       go run agent (lab data dir)"
	@echo "  make run-worker      go run worker (optional)"
	@echo ""
	@echo "  Other:"
	@echo "  make test-pkgs       Test shared Go packages"
	@echo "  make tidy            go mod tidy (all modules)"
	@echo "  make web-install     npm install in web/"
	@echo "  make web-dev         Vite dev server (alias of run-web)"

dev-up:
	docker compose -f docker-compose.dev.yml up -d

dev-down:
	docker compose -f docker-compose.dev.yml down

migrate-status:
	$(GOOSE) -dir $(MIGRATE_DIR) postgres "$(SG_DB_URL)" status

migrate-up:
	$(GOOSE) -dir $(MIGRATE_DIR) postgres "$(SG_DB_URL)" up

migrate-down:
	$(GOOSE) -dir $(MIGRATE_DIR) postgres "$(SG_DB_URL)" down

test-pkgs:
	cd pkgs && go test ./...

build-api:
	cd services/controlplane && go build -o ../../bin/sailguard-api ./cmd/api

build-worker:
	cd services/controlplane && go build -o ../../bin/sailguard-worker ./cmd/worker

build-agent:
	cd agent && go build -o ../bin/sailguard-agent ./cmd/sailguard-agent

build-agent-win7:
	./scripts/build-agent-win7.sh

build-agent-windows:
	./scripts/build-agent-windows.sh

build-agent-setup:
	./scripts/build-agent-setup.sh

tidy:
	cd pkgs && go mod tidy
	cd services/controlplane && go mod tidy
	cd agent && go mod tidy

web-install:
	cd web && npm install

# Existing name kept; run-web is the A/B/C-friendly alias.
web-dev:
	cd web && npm run dev

run-web: web-dev

# Load .env into the process environment when present (SG_HTTP_ADDR, DB, Redis, …).
run-api:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run ./services/controlplane/cmd/api

run-worker:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run ./services/controlplane/cmd/worker

run-agent:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	SG_DATA_DIR="$${SG_DATA_DIR:-$(SG_DATA_DIR)}" \
	SG_SERVER_URL="$${SG_SERVER_URL:-$(SG_SERVER_URL)}" \
	go run ./agent/cmd/sailguard-agent
