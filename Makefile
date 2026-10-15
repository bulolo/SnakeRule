.PHONY: help dev-init check-dev-env dev dev-up dev-down dev-backend run-backend build-backend format check-backend check-all test infra-up infra-down infra-logs docker-up docker-down

GO ?= go
COMPOSE = docker compose -f docker-compose.dev.yml
SERVER_DIR = backend/server
SERVER_BIN = backend/bin/snakerule-server

help:
	@echo "SnakeRule development commands"
	@echo "  make dev-init       Create backend/.env from the example (keeps existing config)"
	@echo "  make dev-up         Run API + rule editor at http://localhost:8080"
	@echo "  make dev-down       Stop the foreground dev server with Ctrl+C"
	@echo "  make run-backend    Alias for running the API server"
	@echo "  make build-backend  Build backend/bin/snakerule-server"
	@echo "  make format         Format Go source"
	@echo "  make check-backend  Check formatting, vet, and compile the backend"
	@echo "  make test           Run backend Go tests"
	@echo "  make infra-up       Start optional PostgreSQL and Redis containers"
	@echo "  make infra-down     Stop optional infrastructure containers"
	@echo "  make infra-logs     Follow optional infrastructure logs"

dev-init:
	@if [ -f backend/.env ]; then \
		echo "backend/.env already exists; leaving it unchanged"; \
	else \
		cp backend/.env.example backend/.env; \
		echo "Created backend/.env"; \
	fi

check-dev-env:
	@if [ ! -f backend/.env ]; then \
		echo "backend/.env not found; run 'make dev-init' first"; \
		exit 1; \
	fi

dev: dev-up
dev-up: dev-init
dev-up:
	@echo "SnakeRule UI and API: http://localhost:8080 (press Ctrl+C to stop)"
	$(MAKE) --no-print-directory run-backend
dev-down:
	@echo "dev-up runs in the foreground; press Ctrl+C in that terminal to stop it."
dev-backend: run-backend
run-backend: check-dev-env
	cd $(SERVER_DIR) && $(GO) run ./cmd/main

build-backend:
	mkdir -p backend/bin
	cd $(SERVER_DIR) && $(GO) build -o ../bin/snakerule-server ./cmd/main

format:
	find backend/server -name '*.go' -type f -print0 | xargs -0 gofmt -w

check-backend:
	@test -z "$$(gofmt -l $$(find backend/server -name '*.go' -type f))" || (echo "Run 'make format' to format Go files"; exit 1)
	cd $(SERVER_DIR) && $(GO) vet ./... && $(GO) build ./...

check-all: check-backend

test:
	cd $(SERVER_DIR) && $(GO) test ./...

# PostgreSQL and Redis are optional at this stage; the API uses in-memory rulesets.
infra-up docker-up:
	$(COMPOSE) up -d postgres redis

infra-down docker-down:
	$(COMPOSE) down

infra-logs:
	$(COMPOSE) logs -f postgres redis
