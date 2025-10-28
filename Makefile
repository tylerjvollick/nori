DOCKER_COMPOSE_DEV = ./docker/docker-compose.dev.yml

dev:
	docker compose -f $(DOCKER_COMPOSE_DEV) up --remove-orphans || make dev-down

dev-down:
	docker compose -f $(DOCKER_COMPOSE_DEV) down --remove-orphans
dev-fresh:
	docker compose -f $(DOCKER_COMPOSE_DEV) down -v --remove-orphans 
	
	
dev-update:
	docker compose -f $(DOCKER_COMPOSE_DEV) up --build -V --remove-orphans

dev-server:
	docker compose -f $(DOCKER_COMPOSE_DEV) up --build -d nori-server --remove-orphans

dev-web:
	docker compose -f $(DOCKER_COMPOSE_DEV) up nori-web --remove-orphans

# migrate -path ./migrations -database "postgres://postgres:password@localhost:5432/nori?sslmode=disable" up
dev-db:
	docker compose -f $(DOCKER_COMPOSE_DEV) up -d database --remove-orphans || make dev-db-down

open-api:
	cd ./open-api && bash ./bin/generate-open-api.sh

