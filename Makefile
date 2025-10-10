ENV ?= development
ENV_FILE := .env.$(ENV)

.PHONY: help dev dev-watch db-up db-down db-reset docker-up docker-down docker-build \
        docker-logs code-check clean install-tools

help:
	@echo "Available commands:"
	@echo ""
	@echo "Code Quality:"
	@echo "  install-tools - Install required Go tools"
	@echo "  code-check    - Format and lint code (run before commit)"
	@echo ""
	@echo "Development:"
	@echo "  dev           - Run the application locally (ENV=dev default)"
	@echo "  dev-watch     - Run with hot reload (ENV=dev default)"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Database:"
	@echo "  db-up         - Start databases (ENV=dev default)"
	@echo "  db-down       - Stop databases"
	@echo "  db-reset      - Reset databases (removes all data)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-up     - Start all services (ENV=production for production)"
	@echo "  docker-down   - Stop all services"
	@echo "  docker-build  - Rebuild API image"
	@echo "  docker-logs   - Show logs"

GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

install-tools:
	@echo "📦 Installing Go tools..."
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/air-verse/air@latest
	@echo "✅ Tools installed!"

code-check:
	@echo "🔍 Formatting and linting..."
	@$(GOBIN)/gofumpt -l -w .
	@$(GOBIN)/golangci-lint run --fix ./...
	@echo "✅ Code check complete!"

clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@go clean -cache
	@echo "✅ Done!"

dev:
	@echo "🚀 Starting application with $(ENV) environment..."
	@set -a && . $(ENV_FILE) && set +a && go run cmd/main/main.go

dev-watch:
	@echo "🔄 Starting with hot reload ($(ENV) environment)..."
	@ENV_FILE=$(ENV_FILE) $(GOBIN)/air -c .air.toml

docker-up:
	@echo "🚀 Starting all services with $(ENV) environment..."
	@docker-compose --env-file $(ENV_FILE) up -d
	@sleep 10
	@echo "✅ Running at http://localhost:8080"

docker-down:
	@echo "⏹️  Stopping services..."
	@docker-compose --env-file $(ENV_FILE) down

docker-build:
	@echo "🏗️  Building API image ($(ENV) environment)..."
	@docker-compose --env-file $(ENV_FILE) build api

docker-logs:
	@echo "📋 Showing logs..."
	@docker-compose --env-file $(ENV_FILE) logs -f api

db-up:
	@echo "🗄️  Starting databases ($(ENV) environment)..."
	@docker-compose --env-file $(ENV_FILE) up -d postgres mongodb
	@sleep 5
	@echo "✅ Databases ready!"

db-down:
	@echo "⏹️  Stopping databases..."
	@docker-compose --env-file $(ENV_FILE) stop postgres mongodb

db-reset:
	@echo "🔄 Resetting databases ($(ENV) environment)..."
	@docker-compose --env-file $(ENV_FILE) down -v
	@docker-compose --env-file $(ENV_FILE) up -d postgres mongodb
	@sleep 5
	@echo "✅ Databases reset!"
