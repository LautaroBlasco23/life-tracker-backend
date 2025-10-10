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
	@echo "  dev           - Run the application locally"
	@echo "  dev-watch     - Run with hot reload"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Database:"
	@echo "  db-up         - Start databases"
	@echo "  db-down       - Stop databases"
	@echo "  db-reset      - Reset databases (removes all data)"
	@echo ""
	@echo "Docker:"
	@echo "  docker-up     - Start all services"
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
	@echo "🚀 Starting application..."
	@go run cmd/main/main.go

dev-watch:
	@echo "🔄 Starting with hot reload..."
	@$(GOBIN)/air -c .air.toml

db-up:
	@echo "🗄️  Starting databases..."
	@docker-compose up -d postgres mongodb
	@sleep 5
	@echo "✅ Databases ready!"

db-down:
	@echo "⏹️  Stopping databases..."
	@docker-compose stop postgres mongodb

db-reset:
	@echo "🔄 Resetting databases..."
	@docker-compose down -v
	@docker-compose up -d postgres mongodb
	@sleep 5
	@echo "✅ Databases reset!"

docker-build:
	@echo "🏗️  Building API image..."
	@docker-compose build api

docker-up:
	@echo "🚀 Starting all services..."
	@docker-compose up -d
	@sleep 10
	@echo "✅ Running at http://localhost:8080"

docker-down:
	@echo "⏹️  Stopping services..."
	@docker-compose down

docker-logs:
	@echo "📋 Showing logs..."
	@docker-compose logs -f api
