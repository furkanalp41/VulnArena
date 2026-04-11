.PHONY: dev dev-deps api migrate-up migrate-down seed test lint build build-cli build-all clean

# Start infrastructure (Postgres + Redis)
dev-deps:
	docker compose up -d

# Stop infrastructure
dev-deps-down:
	docker compose down

# Run API server
api:
	go run ./cmd/api

# Run migrations up
migrate-up:
	go run ./cmd/migrate -direction up

# Run migrations down
migrate-down:
	go run ./cmd/migrate -direction down

# Seed database with challenges and academy content
seed:
	go run ./cmd/seed

# Run all tests
test:
	go test ./... -v -race

# Lint
lint:
	golangci-lint run ./...

# Build API binary
build:
	go build -o bin/api ./cmd/api

# Build CLI binary
build-cli:
	go build -o bin/vulnarena ./cmd/cli

# Build all binaries
build-all: build build-cli

# Clean build artifacts
clean:
	rm -rf bin/

# Full dev setup: start deps, migrate, seed, run API
dev: dev-deps migrate-up seed api
