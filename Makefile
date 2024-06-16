include .env

MIGRATION_NAME = $(word 2, $(MAKECMDGOALS))
DATABASE_URL = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_DATABASE)?sslmode=disable
DOCKER_NETWORK = host

run:
	go run main.go

test:
	go test ./track/...

migration:
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(MIGRATION_NAME)

migrate-up:
	migrate -path=$(MIGRATIONS_DIR) -database $(DATABASE_URL) up $(N)

migrate-down:
	migrate -path=$(MIGRATIONS_DIR) -database $(DATABASE_URL) down $(N)

test-db-up:
	cd deploy/develop && docker compose up -d

test-db-down:
	cd deploy/develop && docker compose down

.PHONY: migration test-db-up test-db-down
