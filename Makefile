DOCKER_COMPOSE_DEV = ./docker/docker-compose.dev.yml
DOCKER_ENV_FILE    = ./docker/.env
DC_DEV             = docker compose -f $(DOCKER_COMPOSE_DEV) --env-file $(DOCKER_ENV_FILE)

dev:
	$(DC_DEV) up --build --remove-orphans || make dev-down

dev-down:
	$(DC_DEV) down --remove-orphans
dev-fresh:
	$(DC_DEV) down -v --remove-orphans 
	
dev-update:
	$(DC_DEV) up --build -V --remove-orphans

dev-server:
	$(DC_DEV) up --build -d nori-server --remove-orphans

dev-web:
	$(DC_DEV) up --build nori-web --remove-orphans

dev-db:
	$(DC_DEV) up -d database --remove-orphans || make dev-db-down

# Migration commands (using embedded nori binary)
migrate-up:
	$(LOAD_ENV) && cd server && go run . migrate up

migrate-down:
	$(LOAD_ENV) && migrate -path ./server/migrations -database "postgres://$$DB_USER:$$DB_PASSWORD@localhost:$$DB_PORT/$$DB_NAME?sslmode=disable" down 1

migrate-status:
	$(LOAD_ENV) && migrate -path ./server/migrations -database "postgres://$$DB_USER:$$DB_PASSWORD@localhost:$$DB_PORT/$$DB_NAME?sslmode=disable" version

migrate-force:
	@echo "Usage: make migrate-force VERSION=<version>"
	@if [ -z "$(VERSION)" ]; then echo "ERROR: VERSION is required"; exit 1; fi
	$(LOAD_ENV) && migrate -path ./server/migrations -database "postgres://$$DB_USER:$$DB_PASSWORD@localhost:$$DB_PORT/$$DB_NAME?sslmode=disable" force $(VERSION)

# Docker cleanup commands
docker-clean:
	docker system prune -f

docker-clean-all:
	docker system prune -af --volumes

docker-stats:
	docker system df

# --- Native dev workflow (no Docker for Go server) ---
# Requires: make dev-db (Postgres in Docker), air (go install github.com/air-verse/air@latest)

# Load docker/.env vars and remap POSTGRES_* → DB_* for the Go server.
# We source the file inside a subshell with 'set -a' but wrap values so
# spaces are preserved, then export only the vars the server needs.
define LOAD_ENV
	eval $$(grep -v '^\s*\#' ./docker/.env | grep -v '^\s*$$' | sed 's/=\(.*\)/="\1"/' | sed 's/^/export /') && \
	export DB_HOST=localhost \
	       DB_USER="$${POSTGRES_USER:-postgres}" \
	       DB_PASSWORD="$${POSTGRES_PASSWORD}" \
	       DB_NAME="$${POSTGRES_DB:-nori}" \
	       DB_PORT="$${DB_HOST_PORT:-5433}" \
	       NORI_PORT="$${NORI_HOST_PORT:-8081}"
endef

# Run migrations natively (against local Postgres)
migrate-native:
	$(LOAD_ENV) && cd server && go run . migrate up

# Start the Go server with air (live reload)
serve:
	$(LOAD_ENV) && $(HOME)/go/bin/air

# One-shot: start Postgres, run migrations, then start air
dev-local:
	$(MAKE) dev-db
	@echo "Waiting for Postgres to be healthy..."
	@until docker exec nori-db pg_isready -U postgres > /dev/null 2>&1; do sleep 1; done
	$(MAKE) migrate-native
	$(MAKE) serve

open-api:
	cd ./open-api && bash ./bin/generate-open-api.sh

