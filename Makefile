.PHONY: build run test clean help coverage docker-up docker-down docker-logs setup-db

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

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) coverage.out logs/

docker-up: ## Start services with docker-compose
	docker-compose up -d

docker-down: ## Stop services
	docker-compose down

docker-logs: ## Follow service logs
	docker-compose logs -f

setup-db: ## Initialize ClickHouse database and tables
	@if [ ! -f .env ]; then echo "WARNING: .env file not found, using defaults"; fi
	set -a && . ./.env && set +a && go run ./cmd/setup-db
