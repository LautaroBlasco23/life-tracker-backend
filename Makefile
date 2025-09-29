# Golang REST API Makefile
.PHONY: dev dev-watch db-up db-down db-reset docker-up docker-down docker-build docker-logs help

# Default target
help:
	@echo "Available commands:"
	@echo ""
	@echo "Development Commands:"
	@echo "  dev             - Run the Go application locally"
	@echo "  dev-watch       - Run with hot reload (restarts on file changes)"
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

# Local development commands
dev:
	@echo "Starting Go application locally..."
	go run cmd/main/main.go

dev-watch:
	@echo "Starting Go application with hot reload..."
	@if ! command -v air > /dev/null 2>&1; then \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	$(shell go env GOPATH)/bin/air -c .air.toml

# Database only commands (for local development)
db-up:
	@echo "Starting databases only..."
	docker-compose up -d postgres mongodb
	@echo "Waiting for databases to be ready..."
	@sleep 10
	@echo "✅ Databases are ready!"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 MongoDB: localhost:27017"

db-down:
	@echo "Stopping databases..."
	docker-compose stop postgres mongodb
	@echo "✅ Databases stopped!"

db-reset:
	@echo "Resetting databases..."
	docker-compose down postgres mongodb
	docker-compose down -v
	@echo "Starting fresh databases..."
	docker-compose up -d postgres mongodb
	@echo "Waiting for databases to be ready..."
	@sleep 15
	@echo "✅ Databases reset complete!"

# Docker commands (full stack)
docker-build:
	@echo "Building API Docker image..."
	docker-compose build api
	@echo "✅ API image built successfully!"

docker-up:
	@echo "Starting all services (API + databases)..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 20
	@echo "✅ All services are running!"
	@echo "🚀 API: http://localhost:8080"
	@echo "🏥 Health check: http://localhost:8080/health"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 MongoDB: localhost:27017"
	@echo "🌐 Mongo Express: http://localhost:8081 (admin/admin123)"

docker-down:
	@echo "Stopping all services..."
	docker-compose down
	@echo "✅ All services stopped!"

docker-logs:
	@echo "Showing logs for all services..."
	docker-compose logs -f

docker-logs-api:
	@echo "Showing API logs..."
	docker-compose logs -f api

docker-restart:
	@echo "Restarting API service..."
	docker-compose restart api
	@echo "✅ API service restarted!"

docker-clean:
	@echo "Cleaning up Docker resources..."
	docker-compose down -v --remove-orphans
	docker system prune -f
	@echo "✅ Docker cleanup complete!"

# Quick shortcuts
up: docker-up
down: docker-down
logs: docker-logs
build: docker-build
