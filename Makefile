.PHONY: help install-tools code-check dev dev-watch prod-up prod-down prod-build db-up db-down db-remove test test-unit test-integration test-coverage test-bench
.DEFAULT_GOAL := help

GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

DOCKER_NETWORK := life-tracker-test-network
MONGO_CONTAINER := life-tracker-test-mongo
POSTGRES_CONTAINER := life-tracker-test-postgres
MONGO_PORT := 27017
POSTGRES_PORT := 5432
MONGO_URI := mongodb://localhost:$(MONGO_PORT)
POSTGRES_DSN := host=localhost user=testuser password=testpass dbname=testdb port=$(POSTGRES_PORT) sslmode=disable

help:
	@echo ""
	@echo "  🛠️  Development:"
	@echo "    install-tools      - Install Go tools (gofumpt, golangci-lint, air, gotestsum)"
	@echo "    code-check         - Format and lint code"
	@echo "    dev                - Start application"
	@echo "    dev-watch          - Start with hot reload"
	@echo ""
	@echo "  🧪 Testing:"
	@echo "    test               - Run all tests"
	@echo "    test-unit          - Run unit tests"
	@echo "    test-integration   - Run integration tests"
	@echo "    test-coverage      - Generate coverage report"
	@echo "    test-bench         - Run benchmarks"
	@echo ""
	@echo "  🐳 Production:"
	@echo "    prod-up            - Start all services"
	@echo "    prod-down          - Stop services"
	@echo "    prod-build         - Build API image"
	@echo ""
	@echo "  🗄️  Database:"
	@echo "    db-up              - Start databases"
	@echo "    db-down            - Stop databases"
	@echo "    db-remove          - Remove databases and volumes"

install-tools:
	@echo "📦 Installing Go tools..."
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/air-verse/air@latest
	@go install gotest.tools/gotestsum@latest
	@echo "✅ Tools installed!"

code-check:
	@echo "🔍 Formatting and linting..."
	@$(GOBIN)/gofumpt -l -w .
	@$(GOBIN)/golangci-lint run --fix ./...
	@echo "✅ Code check complete!"

dev:
	@echo "🚀 Starting application..."
	@go run cmd/main/main.go

dev-watch:
	@echo "🔄 Starting with hot reload..."
	@ENV_FILE=.env $(GOBIN)/air -c .air.toml

prod-up:
	@[ -f .env ] || (echo "❌ .env not found"; exit 1)
	@echo "🚀 Starting all services..."
	@docker-compose --env-file .env up -d
	@sleep 10
	@echo "✅ Running at http://localhost:8080"

prod-down:
	@echo "⏹️  Stopping services..."
	@docker-compose down

prod-build:
	@[ -f .env ] || (echo "❌ .env not found"; exit 1)
	@echo "🏗️  Building API image..."
	@docker-compose build api

db-up:
	@echo "🗄️  Starting databases..."
	@docker-compose -f docker-compose.db.yml up -d
	@sleep 5
	@echo "✅ Databases ready!"

db-down:
	@echo "⏹️  Stopping databases..."
	@docker-compose -f docker-compose.db.yml stop

db-remove:
	@echo "🔄 Removing databases..."
	@docker-compose -f docker-compose.db.yml down -v
	@echo "✅ Databases removed!"

_start-containers:
	@echo "🐳 Starting test containers..."
	@docker network create $(DOCKER_NETWORK) 2>/dev/null || true
	@docker run -d --rm --name $(MONGO_CONTAINER) --network $(DOCKER_NETWORK) -p $(MONGO_PORT):27017 mongo:7 >/dev/null 2>&1 || true
	@docker run -d --rm --name $(POSTGRES_CONTAINER) --network $(DOCKER_NETWORK) -p $(POSTGRES_PORT):5432 \
		-e POSTGRES_USER=testuser -e POSTGRES_PASSWORD=testpass -e POSTGRES_DB=testdb postgres:16-alpine >/dev/null 2>&1 || true
	@sleep 4
	@echo "✅ Test databases ready!"

_stop-containers:
	@docker stop $(MONGO_CONTAINER) $(POSTGRES_CONTAINER) 2>/dev/null || true
	@docker network rm $(DOCKER_NETWORK) 2>/dev/null || true

test: _start-containers
	@echo "🧪 Running all tests..."
	@export TEST_MONGO_URI="$(MONGO_URI)" TEST_POSTGRES_DSN="$(POSTGRES_DSN)" && \
	 $(GOBIN)/gotestsum --format testname -- -race -p 1 ./... || (make _stop-containers && exit 1)
	@make _stop-containers
	@echo "✅ All tests completed!"

test-unit: _start-containers
	@echo "🧪 Running unit tests..."
	@export TEST_MONGO_URI="$(MONGO_URI)" TEST_POSTGRES_DSN="$(POSTGRES_DSN)" && \
	 $(GOBIN)/gotestsum --format testname -- -race -short ./... || (make _stop-containers && exit 1)
	@make _stop-containers
	@echo "✅ Unit tests completed!"

test-integration: _start-containers
	@echo "🧪 Running integration tests..."
	@export TEST_MONGO_URI="$(MONGO_URI)" TEST_POSTGRES_DSN="$(POSTGRES_DSN)" && \
	 $(GOBIN)/gotestsum --format testname -- -race -run Integration ./... || (make _stop-containers && exit 1)
	@make _stop-containers
	@echo "✅ Integration tests completed!"

test-coverage: _start-containers
	@echo "🧪 Running tests with coverage..."
	@export TEST_MONGO_URI="$(MONGO_URI)" TEST_POSTGRES_DSN="$(POSTGRES_DSN)" && \
	 $(GOBIN)/gotestsum --format testname -- -race -coverprofile=coverage.out -covermode=atomic ./... || (make _stop-containers && exit 1)
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"
	@make _stop-containers
	@echo "✅ Coverage completed!"

test-bench: _start-containers
	@echo "🧪 Running benchmarks..."
	@export TEST_MONGO_URI="$(MONGO_URI)" TEST_POSTGRES_DSN="$(POSTGRES_DSN)" && \
	 go test -v -bench=. -benchmem ./... || (make _stop-containers && exit 1)
	@make _stop-containers
	@echo "✅ Benchmarks completed!"
