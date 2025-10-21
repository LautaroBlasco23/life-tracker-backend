.PHONY: help install-tools code-check dev dev-watch prod-up prod-down prod-build db-up db-down db-remove
GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

help:
	@echo "📋 Available commands:"
	@echo ""
	@echo "  install-tools      Install required Go tools (gofumpt, golangci-lint, air)"
	@echo "  code-check         Format and lint code"
	@echo "  dev                Start application in development mode"
	@echo "  dev-watch          Start application with hot reload"
	@echo "  prod-up            Start all services in production (docker-compose)"
	@echo "  prod-down          Stop production services"
	@echo "  prod-build         Build production API image"
	@echo "  db-up              Start databases with default values"
	@echo "  db-down            Stop databases"
	@echo "  db-remove          Remove databases and volumes"
	@echo ""

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
dev:
	@echo "🚀 Starting application..."
	@go run cmd/main/main.go
dev-watch:
	@echo "🔄 Starting with hot reload..."
	@ENV_FILE=.env.development $(GOBIN)/air -c .air.toml
prod-up:
	@[ -f .env.production ] || (echo "❌ .env.production not found"; exit 1)
	@echo "🚀 Starting all services with production environment..."
	@docker-compose --env-file .env.production up -d
	@sleep 10
	@echo "✅ Running at http://localhost:8080"
prod-down:
	@echo "⏹️  Stopping services..."
	@docker-compose down
prod-build:
	@[ -f .env.production ] || (echo "❌ .env.production not found"; exit 1)
	@echo "🏗️  Building API image..."
	@docker-compose build api
db-up:
	@echo "🗄️  Starting databases with default values..."
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
