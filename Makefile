AIR_VERSION := v1.66.0

.PHONY: help setup db-diff db-up schema-diff server worker dev

help: ## Show commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "%-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Create .env, download modules, install Air
	@[ -f .env ] || cp .env.example .env
	go mod download
	@command -v air >/dev/null || go install github.com/air-verse/air@$(AIR_VERSION)

db-diff: ## Generate a SQL migration from GORM models.
	@test -n "$(NAME)" || { echo "Usage: make db-diff NAME=initial_schema DATABASE_URL=<url>"; exit 2; }
	@test -n "$(DATABASE_URL)" || { echo "Pass DATABASE_URL=<url>"; exit 2; }
	@mkdir -p migrations
	@atlas migrate diff "$(NAME)" --env gorm --dev-url "$(DATABASE_URL)"

db-up: ## Apply migrations
	go run ./cmd/migrate

server: ## Run API with live reload
	air -c .air.api.toml