.PHONY: help dev-up dev-down test-pkgs build-api build-worker build-agent tidy web-install web-dev migrate-status migrate-up migrate-down

# Prefer goose on PATH; fall back to $(go env GOPATH)/bin/goose
GOOSE ?= $(shell command -v goose 2>/dev/null || echo $$(go env GOPATH)/bin/goose)
SG_DB_URL ?= postgres://sailguard:sailguard@127.0.0.1:15433/sailguard?sslmode=disable
MIGRATE_DIR := services/controlplane/migrations

help:
	@echo "SailGuard targets:"
	@echo "  make dev-up          Start Postgres + Redis (dev)"
	@echo "  make dev-down        Stop dev dependencies"
	@echo "  make migrate-status  Goose migration status"
	@echo "  make migrate-up      Apply pending migrations"
	@echo "  make migrate-down    Roll back one migration"
	@echo "  make test-pkgs       Test shared Go packages"
	@echo "  make build-api       Build control plane API"
	@echo "  make build-worker    Build worker"
	@echo "  make build-agent     Build agent for current GOOS"
	@echo "  make tidy            go mod tidy (all modules)"
	@echo "  make web-install     npm install in web/"
	@echo "  make web-dev         Vite dev server"

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

tidy:
	cd pkgs && go mod tidy
	cd services/controlplane && go mod tidy
	cd agent && go mod tidy

web-install:
	cd web && npm install

web-dev:
	cd web && npm run dev
