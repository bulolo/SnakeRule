.PHONY: help dev-init check-dev-env dev dev-up dev-open dev-down dev-clean dev-build dev-rebuild dev-restart dev-restart-backend dev-logs dev-logs-backend dev-logs-admin dev-logs-nginx dev-logs-docs dev-status dev-backend run-backend build-backend format check-backend check-all test gen-sdk set-version docs-install docs-dev docs-build docs-open infra-up infra-down infra-logs docker-up docker-down prod-init check-prod-env prod-build prod-up prod-down helm-lint helm-template helm-install helm-uninstall publish-github publish-codeup publish-all

GO ?= go
DEV_COMPOSE = docker compose --env-file backend/.env -f docker-compose.dev.yml
PROD_COMPOSE = docker compose -p snakerule-prod --project-directory . --env-file deploy/.env.prod -f deploy/docker-compose.prod.yml
SERVER_DIR = backend/server
APP_PORT ?= $(or $(SERVER_PORT),$(shell sed -n 's/^SERVER_PORT=//p' backend/.env 2>/dev/null | tail -n 1),8080)
ADMIN_PORT ?= $(or $(ADMIN_PORT_OVERRIDE),$(shell sed -n 's/^ADMIN_PORT=//p' backend/.env 2>/dev/null | tail -n 1),3000)
GATEWAY_PORT ?= $(or $(shell sed -n 's/^GATEWAY_PORT=//p' backend/.env 2>/dev/null | tail -n 1),8080)
DOCS_PORT ?= $(or $(shell sed -n 's/^DOCS_PORT=//p' backend/.env 2>/dev/null | tail -n 1),8003)
GITHUB_URL ?= https://github.com/bulolo/SnakeRule.git
CODEUP_REMOTE ?= origin
PUBLISH_BRANCH ?= $(shell git branch --show-current)
PUSH_DRY_RUN = $(if $(filter 1,$(DRY_RUN)),--dry-run,)
VERSION_FILE = VERSION
SDK_SPEC = backend/server/docs/openapi.json
SDK_OUTPUT = frontend/admin/src/lib/sdk/generated.ts
HELM ?= helm
HELM_CHART = deploy/helm
HELM_RELEASE ?= snakerule
HELM_NAMESPACE ?= snakerule
HELM_VALUES ?=
HELM_SET ?=
HELM_VALUES_ARG = $(if $(HELM_VALUES),-f $(HELM_VALUES),)

help:
	@echo "SnakeRule development commands"
	@echo "  make dev-init       Create backend/.env from the example (keeps existing config)"
	@echo "  make dev-up         Start Nginx + admin frontend + API (http://localhost:$(GATEWAY_PORT))"
	@echo "                      Docs: http://localhost:$(GATEWAY_PORT)/docs/"
	@echo "  make dev-open       Open the running admin frontend in a browser"
	@echo "  make docs-open      Open the documentation site in a browser"
	@echo "  make docs-dev       Run the documentation site locally (port 8003)"
	@echo "  make docs-build     Build the static documentation site"
	@echo "  make dev-down       Stop and remove dev containers"
	@echo "  make dev-clean      Reset dev containers, images, and data volumes"
	@echo "  make dev-build      Build development images"
	@echo "  make dev-rebuild    Rebuild and start development containers"
	@echo "  make dev-restart    Restart the development stack"
	@echo "  make dev-logs       Follow all development logs"
	@echo "  make dev-logs-backend Follow API logs"
	@echo "  make dev-logs-admin Follow admin frontend logs"
	@echo "  make dev-logs-nginx Follow Nginx reverse proxy logs"
	@echo "  make dev-logs-docs  Follow documentation site logs"
	@echo "  make dev-status     Show container status and frontend/API health"
	@echo "  make prod-init      Create production Compose settings and private secrets"
	@echo "  make prod-build     Build production API, admin, and docs images"
	@echo "  make prod-up        Start the production Compose stack in the background"
	@echo "  make prod-down      Stop the production Compose stack"
	@echo "  make helm-lint      Lint the Kubernetes chart"
	@echo "  make helm-template  Render the chart locally (HELM_VALUES=path.yaml)"
	@echo "  make helm-install   Install/upgrade with Helm in namespace $(HELM_NAMESPACE)"
	@echo "  make helm-uninstall Remove the Helm release (PVC data is retained)"
	@echo "  make run-backend    Alias for running the API server"
	@echo "  make build-backend  Build backend/bin/snakerule-server"
	@echo "  make format         Format Go source"
	@echo "  make check-backend  Check formatting, vet, and compile the backend"
	@echo "  make test           Run backend Go tests"
	@echo "  make gen-sdk        Generate TypeScript API types from $(SDK_SPEC)"
	@echo "  make set-version v=0.1.1 Update the project version"
	@echo "  make infra-up       Start optional PostgreSQL and Redis containers"
	@echo "  make infra-down     Stop optional infrastructure containers"
	@echo "  make infra-logs     Follow optional infrastructure logs"
	@echo "  make publish-github [DRY_RUN=1] Push current branch to $(GITHUB_URL)"
	@echo "  make publish-codeup [DRY_RUN=1] Push current branch to Codeup ($(CODEUP_REMOTE))"
	@echo "  make publish-all [DRY_RUN=1]    Push current branch to both remotes"

dev-init:
	python3 scripts/init_dev_env.py

check-dev-env:
	@if [ ! -f backend/.env ]; then \
		echo "backend/.env not found; run 'make dev-init' first"; \
		exit 1; \
	fi

dev: dev-up
dev-build: check-dev-env
	$(DEV_COMPOSE) build
dev-up: check-dev-env
	@echo "SnakeRule Admin: http://localhost:$(GATEWAY_PORT)"
	@echo "SnakeRule Docs:  http://localhost:$(GATEWAY_PORT)/docs/"
	@echo "Direct Admin:    http://localhost:$(ADMIN_PORT)"
	@echo "Direct Docs:     http://localhost:$(DOCS_PORT)/docs/"
	@echo "SnakeRule API:   http://localhost:$(APP_PORT)"
	$(DEV_COMPOSE) up --build
dev-open:
	@python3 -c 'import webbrowser; webbrowser.open("http://localhost:$(GATEWAY_PORT)")'
docs-install:
	cd frontend/docs && npm install
docs-dev:
	cd frontend/docs && npm run dev
docs-build:
	cd frontend/docs && npm run build
docs-open:
	@python3 -c 'import webbrowser; webbrowser.open("http://localhost:$(GATEWAY_PORT)/docs/")'
prod-init:
	python3 scripts/init_prod_env.py
check-prod-env:
	@if [ ! -f deploy/.env.prod ]; then \
		echo "deploy/.env.prod not found; run 'make prod-init' first"; \
		exit 1; \
	fi
prod-build: check-prod-env
	$(PROD_COMPOSE) build
prod-up: check-prod-env
	$(PROD_COMPOSE) up -d --build
prod-down: check-prod-env
	$(PROD_COMPOSE) down

helm-lint:
	$(HELM) lint $(HELM_CHART) --set-string backend.env.authPassword=lint-only --set-string backend.env.licenseEncryptionKey=lint-only --set-string backend.env.jwtSecret=lint-only

helm-template:
	$(HELM) template $(HELM_RELEASE) $(HELM_CHART) -n $(HELM_NAMESPACE) $(HELM_VALUES_ARG) $(HELM_SET)

helm-install:
	$(HELM) upgrade --install $(HELM_RELEASE) $(HELM_CHART) -n $(HELM_NAMESPACE) --create-namespace $(HELM_VALUES_ARG) $(HELM_SET) --wait

helm-uninstall:
	$(HELM) uninstall $(HELM_RELEASE) -n $(HELM_NAMESPACE)
dev-rebuild: check-dev-env
	$(DEV_COMPOSE) up -d --build --force-recreate
dev-down:
	$(DEV_COMPOSE) --profile infra down --remove-orphans
dev-clean: check-dev-env
	@echo "WARNING: this removes development containers, local images, and all development data volumes."
	@read -p "Type 'yes' to continue: " ans && [ "$${ans}" = "yes" ] || (echo "Cancelled"; exit 1)
	$(DEV_COMPOSE) --profile infra down -v --rmi local --remove-orphans
dev-restart:
	$(DEV_COMPOSE) restart
dev-restart-backend:
	$(DEV_COMPOSE) restart backend
dev-logs:
	$(DEV_COMPOSE) logs -f
dev-logs-backend:
	$(DEV_COMPOSE) logs -f backend
dev-logs-admin:
	$(DEV_COMPOSE) logs -f admin
dev-logs-nginx:
	$(DEV_COMPOSE) logs -f nginx
dev-logs-docs:
	$(DEV_COMPOSE) logs -f docs
dev-status:
	$(DEV_COMPOSE) --profile infra ps
	@curl -fsS http://localhost:$(GATEWAY_PORT)/ >/dev/null && echo "Admin frontend: http://localhost:$(GATEWAY_PORT)"
	@curl -fsS http://localhost:$(GATEWAY_PORT)/docs/ >/dev/null && echo "Documentation: http://localhost:$(GATEWAY_PORT)/docs/"
	@curl -fsS http://localhost:$(GATEWAY_PORT)/api/v1/meta
dev-backend: check-dev-env
	$(DEV_COMPOSE) up --build backend
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

gen-sdk:
	python3 scripts/generate_sdk.py $(SDK_SPEC) $(SDK_OUTPUT)

set-version:
	python3 scripts/set_version.py "$(v)"

# PostgreSQL and Redis are optional at this stage; the API uses in-memory rulesets.
infra-up docker-up:
	$(DEV_COMPOSE) --profile infra up -d postgres redis

infra-down docker-down:
	$(DEV_COMPOSE) --profile infra stop postgres redis

infra-logs:
	$(DEV_COMPOSE) --profile infra logs -f postgres redis

# Publish the selected local branch to the configured GitHub and Codeup repositories.
# Use DRY_RUN=1 to ask Git to report what it would push without changing either remote.
publish-github:
	@test -n "$(PUBLISH_BRANCH)" || (echo "No current branch; set PUBLISH_BRANCH explicitly"; exit 1)
	git push $(PUSH_DRY_RUN) $(GITHUB_URL) $(PUBLISH_BRANCH)

publish-codeup:
	@test -n "$(PUBLISH_BRANCH)" || (echo "No current branch; set PUBLISH_BRANCH explicitly"; exit 1)
	@git remote get-url $(CODEUP_REMOTE) >/dev/null || (echo "Codeup remote '$(CODEUP_REMOTE)' is not configured"; exit 1)
	git push $(PUSH_DRY_RUN) $(CODEUP_REMOTE) $(PUBLISH_BRANCH)

publish-all: publish-codeup publish-github
