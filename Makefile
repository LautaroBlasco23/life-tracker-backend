.PHONY: help dev dev-watch db-up db-down db-reset docker-up docker-down docker-build docker-logs \
        fmt lint lint-fix vet test test-coverage check clean tidy deps security install-tools \
        pre-commit pre-push

help:
	@echo "Available commands:"
	@echo ""
	@echo "Code Quality Commands:"
	@echo "  install-tools   - Install all required Go tools (gofumpt, golangci-lint, etc.)"
	@echo "  fmt             - Format Go code with gofumpt and goimports"
	@echo "  lint            - Run golangci-lint checks"
	@echo "  lint-fix        - Run golangci-lint with auto-fix"
	@echo "  vet             - Run go vet static analysis"
	@echo "  test            - Run tests with race detection and coverage"
	@echo "  test-coverage   - Run tests and generate HTML coverage report"
	@echo "  check           - Run all checks (fmt, lint, vet, test)"
	@echo "  security        - Run security checks with gosec"
	@echo "  tidy            - Tidy and verify go.mod"
	@echo "  deps            - Download dependencies"
	@echo ""
	@echo "Development Commands:"
	@echo "  dev             - Run the Go application locally"
	@echo "  dev-watch       - Run with hot reload (restarts on file changes)"
	@echo "  build           - Build the application binary"
	@echo "  clean           - Clean build artifacts and test cache"
	@echo ""
	@echo "Database Commands:"
	@echo "  db-up           - Start PostgreSQL and MongoDB database containers only"
	@echo "  db-down         - Stop all database containers"
	@echo "  db-reset        - Reset databases (stop, remove volumes, start fresh)"
	@echo ""
	@echo "Docker Commands (Full Stack):"
	@echo "  docker-build    - Build the API Docker image"
	@echo "  docker-up       - Start all services (API + databases)"
	@echo "  docker-down     - Stop all services"
	@echo "  docker-logs     - Show logs for all services"
	@echo "  docker-logs-api - Show API logs only"
	@echo "  docker-restart  - Restart the API service"
	@echo "  docker-clean    - Clean up containers, networks, and volumes"
	@echo "  docker-api-only - Start only the API container (databases must be running)"
	@echo "  update-deploy   - Rebuild and redeploy API with cleanup (keeps databases)"
	@echo "  reboot          - Complete restart: stop all, remove data, rebuild, start fresh"

GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

install-tools:
	@echo "📦 Installing Go development tools..."
	@go install mvdan.cc/gofumpt@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install github.com/air-verse/air@latest
	@echo "✅ All tools installed successfully!"
	@echo "⚠️  Make sure $(GOBIN) is in your PATH"
	@echo "   Add this to your ~/.bashrc or ~/.zshrc:"
	@echo "   export PATH=\$$PATH:$(GOBIN)"

fmt:
	@echo "🎨 Formatting Go code..."
	@$(GOBIN)/gofumpt -l -w .
	@$(GOBIN)/goimports -w -local life-tracker-backend .
	@echo "✅ Code formatted!"

lint:
	@echo "🔍 Running linters..."
	@$(GOBIN)/golangci-lint run ./...
	@echo "✅ Linting complete!"

lint-fix:
	@echo "🔧 Running linters with auto-fix..."
	@$(GOBIN)/golangci-lint run --fix ./...
	@echo "✅ Linting with fixes complete!"

vet:
	@echo "🔬 Running go vet..."
	@go vet ./...
	@echo "✅ Go vet passed!"

test:
	@echo "🧪 Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -n 1
	@echo "✅ Tests complete!"

test-coverage:
	@echo "🧪 Running tests with coverage report..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | tail -n 1

check: fmt lint vet test
	@echo "✅ All checks passed!"

pre-commit:
	@echo "🎯 Running pre-commit checks..."
	@$(MAKE) fmt
	@$(MAKE) lint-fix
	@$(MAKE) vet
	@echo "✅ Pre-commit checks passed!"

pre-push:
	@echo "🎯 Running pre-push checks..."
	@$(MAKE) test
	@echo "✅ Pre-push checks passed!"

security:
	@echo "🔒 Running security checks..."
	@$(GOBIN)/gosec -quiet ./...
	@echo "✅ Security checks complete!"

tidy:
	@echo "🧹 Tidying go.mod..."
	@go mod tidy
	@go mod verify
	@echo "✅ Dependencies tidied!"

deps:
	@echo "📦 Downloading dependencies..."
	@go mod download
	@echo "✅ Dependencies downloaded!"

build:
	@echo "🏗️  Building application..."
	@go build -o bin/server ./cmd/main/main.go
	@echo "✅ Build complete! Binary: bin/server"

clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/ coverage.out coverage.html
	@go clean -testcache -cache -modcache
	@echo "✅ Cleanup complete!"

dev:
	@echo "🚀 Starting Go application locally..."
	@go run cmd/main/main.go

dev-watch:
	@echo "🔄 Starting Go application with hot reload..."
	@if ! command -v $(GOBIN)/air > /dev/null 2>&1; then \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@$(GOBIN)/air -c .air.toml

db-up:
	@echo "🗄️  Starting databases only..."
	@docker-compose up -d postgres mongodb
	@echo "⏳ Waiting for databases to be ready..."
	@sleep 10
	@echo "✅ Databases are ready!"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 MongoDB: localhost:27017"

db-down:
	@echo "⏹️  Stopping databases..."
	@docker-compose stop postgres mongodb
	@echo "✅ Databases stopped!"

db-reset:
	@echo "🔄 Resetting databases..."
	@docker-compose down postgres mongodb
	@docker-compose down -v
	@echo "🚀 Starting fresh databases..."
	@docker-compose up -d postgres mongodb
	@echo "⏳ Waiting for databases to be ready..."
	@sleep 15
	@echo "✅ Databases reset complete!"

docker-build:
	@echo "🏗️  Building API Docker image..."
	@docker-compose build api
	@echo "✅ API image built successfully!"

docker-up:
	@echo "🚀 Starting all services (API + databases)..."
	@docker-compose up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 20
	@echo "✅ All services are running!"
	@echo "🚀 API: http://localhost:8080"
	@echo "🏥 Health check: http://localhost:8080/health"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 MongoDB: localhost:27017"

docker-down:
	@echo "⏹️  Stopping all services..."
	@docker-compose down
	@echo "✅ All services stopped!"

docker-logs:
	@echo "📋 Showing logs for all services..."
	@docker-compose logs -f

docker-logs-api:
	@echo "📋 Showing API logs..."
	@docker-compose logs -f api

docker-restart:
	@echo "🔄 Restarting API service..."
	@docker-compose restart api
	@echo "✅ API service restarted!"

docker-clean:
	@echo "🧹 Cleaning up Docker resources..."
	@docker-compose down -v --remove-orphans
	@docker system prune -f
	@echo "✅ Docker cleanup complete!"

docker-api-only:
	@echo "🚀 Starting only the API container..."
	@docker-compose up -d api
	@echo "✅ API container started!"
	@echo "🚀 API: http://localhost:8080"
	@echo "🏥 Health check: http://localhost:8080/health"

update-deploy:
	@echo "🔄 Stopping and removing old API container..."
	@-docker-compose stop api
	@-docker-compose rm -f api
	@echo "🗑️  Removing old API image..."
	@-docker rmi golang_api_server:latest
	@echo "🏗️  Building new API image..."
	@docker-compose build api
	@echo "🚀 Starting new API container..."
	@docker-compose up -d api
	@echo "⏳ Waiting for API to be ready..."
	@sleep 10
	@echo "🧹 Cleaning up dangling images..."
	@docker image prune -f
	@echo "✅ Deployment complete!"
	@echo "🚀 API: http://localhost:8080"
	@echo "🏥 Health check: http://localhost:8080/health"

reboot:
	@echo "🔄 Complete system reboot: stopping all services..."
	@docker-compose down -v --remove-orphans
	@echo "🧹 Cleaning up old containers and volumes..."
	@docker system prune -f
	@echo "🔨 Rebuilding API image..."
	@docker-compose build api
	@echo "🚀 Starting all services fresh..."
	@docker-compose up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 25
	@echo "✅ Complete reboot finished!"
	@echo "🚀 API: http://localhost:8080"
	@echo "🏥 Health check: http://localhost:8080/health"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 MongoDB: localhost:27017"

up: docker-up
down: docker-down
logs: docker-logs
