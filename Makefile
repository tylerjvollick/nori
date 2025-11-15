DOCKER_COMPOSE_DEV = ./docker/docker-compose.dev.yml

dev:
	docker compose -f $(DOCKER_COMPOSE_DEV) up --build --remove-orphans || make dev-down

dev-down:
	docker compose -f $(DOCKER_COMPOSE_DEV) down --remove-orphans
dev-fresh:
	docker compose -f $(DOCKER_COMPOSE_DEV) down -v --remove-orphans 
	
dev-update:
	docker compose -f $(DOCKER_COMPOSE_DEV) up --build -V --remove-orphans

dev-server:
	docker compose -f $(DOCKER_COMPOSE_DEV) up --build -d nori-server --remove-orphans

dev-web:
	docker compose -f $(DOCKER_COMPOSE_DEV) up --build nori-web --remove-orphans

dev-db:
	docker compose -f $(DOCKER_COMPOSE_DEV) up -d database --remove-orphans || make dev-db-down

# Migration commands
migrate-up:
	migrate -path ./server/migrations -database "postgres://postgres:password@localhost:5432/nori?sslmode=disable" up

migrate-down:
	migrate -path ./server/migrations -database "postgres://postgres:password@localhost:5432/nori?sslmode=disable" down 1

migrate-status:
	migrate -path ./server/migrations -database "postgres://postgres:password@localhost:5432/nori?sslmode=disable" version

migrate-force:
	@echo "Usage: make migrate-force VERSION=<version>"
	@if [ -z "$(VERSION)" ]; then echo "ERROR: VERSION is required"; exit 1; fi
	migrate -path ./server/migrations -database "postgres://postgres:password@localhost:5432/nori?sslmode=disable" force $(VERSION)

# Docker cleanup commands
docker-clean:
	docker system prune -f

docker-clean-all:
	docker system prune -af --volumes

docker-stats:
	docker system df

open-api:
	cd ./open-api && bash ./bin/generate-open-api.sh

