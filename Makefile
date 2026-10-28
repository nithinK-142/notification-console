AIR_VERSION := v1.66.0

.PHONY: help setup db-diff db-up schema-diff server worker dev admin-create admin-active

help: ## Show commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "%-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Create .env, download modules, and install Air
	@[ -f .env ] || cp .env.example .env
	go mod download
	@command -v air >/dev/null 2>&1 || go install github.com/air-verse/air@$(AIR_VERSION)
	@command -v atlas >/dev/null 2>&1 || echo "Atlas CLI missing; install it from https://atlasgo.io/getting-started"
	@[ -f notification-service-key.json ] || echo "Firebase key missing; save it as notification-service-key.json before running the worker."

db-diff: ## Generate a SQL migration from GORM models.
	@test -n "$(NAME)" || { echo "Usage: make db-diff NAME=initial_schema DATABASE_URL=<url>"; exit 2; }
	@test -n "$(DATABASE_URL)" || { echo "Pass DATABASE_URL=<url>"; exit 2; }
	@mkdir -p migrations
	@atlas migrate diff "$(NAME)" --env gorm --dev-url "$(DATABASE_URL)"

db-up: ## Apply pending PostgreSQL migration
	go run ./cmd/migrate

server: ## Run API with live reload
	air -c .air.api.toml

worker: ## Run notification worker with live reload
	air -c .air.worker.toml

dev: ## Run API and worker together; Ctrl-C stops both
	@set -m; \
	  (air -c .air.api.toml) & api=$$!; \
	  (air -c .air.worker.toml) & worker=$$!; \
	  trap 'kill $$api $$worker 2>/dev/null || true; wait' INT TERM EXIT; \
	  wait
	  
admin-create: ## Create an admin (usage: make admin-create EMAIL=admin@example.com)
	@test -n "$(EMAIL)" || { echo "Usage: make admin-create EMAIL=admin@example.com"; exit 2; }
	go run ./cmd/admin create --email "$(EMAIL)"

admin-active: ## Set admin status (usage: make admin-active EMAIL=... ACTIVE=true|false)
	@test -n "$(EMAIL)" || { echo "Usage: make admin-active EMAIL=... ACTIVE=true|false"; exit 2; }
	@case "$(ACTIVE)" in true|false) ;; *) echo "Pass ACTIVE=true or ACTIVE=false"; exit 2;; esac
	go run ./cmd/admin set-active --email "$(EMAIL)" --active="$(ACTIVE)"
