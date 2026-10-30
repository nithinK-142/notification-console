SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

GO ?= go
PNPM ?= pnpm
AIR ?= air
ATLAS ?= atlas

AIR_VERSION ?= v1.66.0
PNPM_VERSION ?= 9.15.9
WEB_HOST ?= 127.0.0.1
WEB_PORT ?= 5173
API_URL ?= http://localhost:8080

.PHONY: help setup setup-env setup-tools deps dev api server worker worker-once web web-build web-preview build build-go test test-race vet fmt fmt-check check db-diff db-up db-hash admin-create admin-active health clean

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_.-]+:.*## / {printf "%-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Prepare local env files, Go tools, and frontend dependencies
	$(MAKE) setup-env
	$(MAKE) setup-tools
	$(MAKE) deps
	@if [ ! -f notification-service-key.json ]; then \
		echo "Firebase key missing: add notification-service-key.json to run the worker."; \
	fi
	@if ! command -v "$(ATLAS)" >/dev/null 2>&1; then \
		echo "Atlas CLI missing: install it from https://atlasgo.io/getting-started"; \
	fi

setup-env: ## Create local env files without overwriting existing values
	@[ -f .env ] || cp .env.example .env
	@[ -f web/.env ] || cp web/.env.example web/.env
	@echo "Environment files ready. Existing files were left unchanged."

setup-tools: ## Check toolchain and install Air when missing
	@command -v "$(GO)" >/dev/null 2>&1 || { echo "Go is required."; exit 1; }
	@command -v node >/dev/null 2>&1 || { echo "Node.js is required (Vite 7 needs Node 20.19+ or 22.12+)."; exit 1; }
	@node -e 'const [major, minor] = process.versions.node.split(".").map(Number); const supported = (major === 20 && minor >= 19) || (major >= 22 && (major > 22 || minor >= 12)); if (!supported) { console.error(`Node ${process.versions.node} is unsupported. Use Node 20.19+ or 22.12+.`); process.exit(1); }'
	@if ! command -v "$(PNPM)" >/dev/null 2>&1; then \
		if command -v corepack >/dev/null 2>&1; then \
			echo "Activating pnpm $(PNPM_VERSION) through Corepack..."; \
			corepack enable; \
			corepack prepare pnpm@$(PNPM_VERSION) --activate; \
		else \
			echo "pnpm is required. Install pnpm or enable Corepack."; exit 1; \
		fi; \
	fi
	@if ! command -v "$(AIR)" >/dev/null 2>&1; then \
		echo "Installing Air $(AIR_VERSION)..."; \
		$(GO) install github.com/air-verse/air@$(AIR_VERSION); \
	fi
	@echo "Go, Node.js, pnpm, and Air are ready."

deps: ## Download Go modules and install frontend packages from the lockfile
	$(GO) mod download
	$(PNPM) --dir web install --frozen-lockfile

# Development

dev: ## Run API, notification worker, and Vite together; Ctrl-C stops all three
	@command -v "$(AIR)" >/dev/null 2>&1 || { echo "Air missing. Run: make setup"; exit 1; }
	@command -v "$(PNPM)" >/dev/null 2>&1 || { echo "pnpm missing. Run: make setup"; exit 1; }
	@set -Eeuo pipefail; \
		api_pid=; worker_pid=; web_pid=; \
		cleanup() { \
			trap - EXIT; \
			for pid in "$$api_pid" "$$worker_pid" "$$web_pid"; do \
				if [[ -n "$$pid" ]]; then kill "$$pid" 2>/dev/null || true; fi; \
			done; \
		for pid in "$$api_pid" "$$worker_pid" "$$web_pid"; do \
			if [[ -n "$$pid" ]]; then wait "$$pid" 2>/dev/null || true; fi; \
		done; \
	}; \
		trap cleanup EXIT; \
		trap 'exit 130' INT; \
		trap 'exit 143' TERM; \
		$(AIR) -c .air.api.toml & api_pid=$$!; \
		$(AIR) -c .air.worker.toml & worker_pid=$$!; \
		$(PNPM) --dir web dev --host $(WEB_HOST) --port $(WEB_PORT) & web_pid=$$!; \
		echo "API + worker + web started. Vite: http://$(WEB_HOST):$(WEB_PORT)"; \
		wait -n "$$api_pid" "$$worker_pid" "$$web_pid"

api: ## Run the Go API with Air live reload
	$(AIR) -c .air.api.toml

server: api ## Alias for api

worker: ## Run the notification worker with Air live reload
	$(AIR) -c .air.worker.toml

worker-once: ## Process currently claimable notification jobs once and exit
	$(GO) run ./cmd/worker --once

web: ## Run the Vite frontend
	$(PNPM) --dir web dev --host $(WEB_HOST) --port $(WEB_PORT)

web-build: ## Type-check and build the frontend
	$(PNPM) --dir web build

web-preview: ## Preview the production frontend build
	$(PNPM) --dir web preview --host $(WEB_HOST) --port $(WEB_PORT)

# Build and checks

build: build-go web-build ## Build API, worker, and frontend

build-go: ## Compile API and worker binaries into bin/
	@mkdir -p bin
	$(GO) build -o bin/api ./cmd/api
	$(GO) build -o bin/worker ./cmd/worker

test: ## Run all Go tests
	$(GO) test ./...

test-race: ## Run Go tests with the race detector
	$(GO) test -race ./...

vet: ## Run go vet across all Go packages
	$(GO) vet ./...

fmt: ## Format Go source files
	gofmt -w $$(find . -type f -name '*.go' -not -path './.git/*' -not -path './tmp/*' -not -path './bin/*' -print)

fmt-check: ## Check Go formatting without modifying files
	@files="$$(gofmt -l $$(find . -type f -name '*.go' -not -path './.git/*' -not -path './tmp/*' -not -path './bin/*' -print))"; \
	if [ -n "$$files" ]; then echo "Go files need formatting:"; echo "$$files"; exit 1; fi

check: fmt-check vet test web-build ## Run formatting, Go checks, tests, and frontend build

# Database

db-diff: ## Create a migration: make db-diff NAME=add_delivery_index ATLAS_DEV_URL=postgres://...
	@test -n "$(NAME)" || { echo "Pass NAME=<migration_name>"; exit 2; }
	@test -n "$(ATLAS_DEV_URL)" || { echo "Pass ATLAS_DEV_URL=<disposable_postgres_url> (do not use production DATABASE_URL)."; exit 2; }
	@command -v "$(ATLAS)" >/dev/null 2>&1 || { echo "Atlas CLI missing: https://atlasgo.io/getting-started"; exit 1; }
	ATLAS_DEV_URL="$(ATLAS_DEV_URL)" $(ATLAS) migrate diff "$(NAME)" --env gorm

db-up: ## Apply pending migrations using DATABASE_URL from .env
	$(GO) run ./cmd/cli migrate up

db-hash: ## Recalculate the Atlas migration checksum
	$(ATLAS) migrate hash --dir file://migrations

# Admin

admin-create: ## Create an admin: make admin-create EMAIL=admin@example.com
	@test -n "$(EMAIL)" || { echo "Pass EMAIL=admin@example.com"; exit 2; }
	$(GO) run ./cmd/cli admin create --email "$(EMAIL)"

admin-active: ## Set admin status: make admin-active EMAIL=... ACTIVE=true|false
	@test -n "$(EMAIL)" || { echo "Pass EMAIL=admin@example.com"; exit 2; }
	@case "$(ACTIVE)" in true|false) ;; *) echo "Pass ACTIVE=true or ACTIVE=false"; exit 2;; esac
	$(GO) run ./cmd/cli admin set-active --email "$(EMAIL)" --active="$(ACTIVE)"

# Operations

health: ## Check API and PostgreSQL health
	@curl --fail --silent --show-error "$(API_URL)/api/health"; echo

clean: ## Remove generated binaries, Air output, and frontend build, node_modules
	rm -rf bin tmp web/dist web/node_modules
