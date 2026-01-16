.PHONY: build run test clean help coverage docker-up docker-down docker-logs setup-db fmt

APP_NAME = logtopus
CLI_NAME = apikey-gen
BUILD_DIR = build/bin

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the ingestion service and CLI tool
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/ingestor
	go build -o $(BUILD_DIR)/$(CLI_NAME) ./cmd/apikey-gen
	go build -o $(BUILD_DIR)/setup-db ./cmd/setup-db
	go build -o $(BUILD_DIR)/query-service ./cmd/query-service
	cp -r public $(BUILD_DIR)/
	@echo "Build complete. Binaries in $(BUILD_DIR)/"

run: build ## Run the ingestion service locally
	@if [ ! -f .env ]; then echo "WARNING: .env file not found, using defaults"; fi
	./$(BUILD_DIR)/$(APP_NAME)

test: ## Run unit tests
	go test -v ./...

coverage: ## Run tests with coverage and display report
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "Coverage report generated: coverage.out"

fmt: ## Format all go code
	go fmt ./...

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) coverage.out logs/

# Docker Mode: clickhouse (default) or file
MODE ?= clickhouse

# Validate MODE
ifeq ($(filter $(MODE),file clickhouse),)
$(error MODE must be 'file' or 'clickhouse')
endif

COMPOSE_FILE = docker-compose.$(MODE).yml

docker-up: ## Start services with docker-compose (MODE=file or MODE=clickhouse) [Default: MODE=clickhouse]
	@echo "Starting in $(MODE) mode using $(COMPOSE_FILE)..."
	docker-compose -f $(COMPOSE_FILE) up -d --build

docker-down: ## Stop services
	@echo "Stopping $(MODE) mode..."
	docker-compose -f $(COMPOSE_FILE) down

docker-logs: ## Follow service logs
	docker-compose -f $(COMPOSE_FILE) logs -f

setup-db: ## Initialize ClickHouse database and tables
	@if [ ! -f .env ]; then echo "WARNING: .env file not found, using defaults"; fi
	set -a && . ./.env && set +a && go run ./cmd/setup-db

apikey: ## Generate API Key (usage: make apikey CLIENT=... SECRET=...)
	@go run ./cmd/apikey-gen -client $(CLIENT) -secret $(SECRET)
