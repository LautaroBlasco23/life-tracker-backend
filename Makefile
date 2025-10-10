.PHONY: dev dev-watch db-up db-down db-reset docker-up docker-down docker-build docker-logs help

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
	@echo "  docker-api-only - Start only the API container (databases must be running)"
	@echo "  update-deploy   - Rebuild and redeploy API with cleanup (keeps databases)"
	@echo "  reboot          - Complete restart: stop all, remove data, rebuild, start fresh"

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

docker-api-only:
	@echo "Starting only the API container..."
	docker-compose up -d api
	@echo "✅ API container started!"
	@echo "🚀 API: http://localhost:8080"
	@echo "🏥 Health check: http://localhost:8080/health"

update-deploy:
	@echo "Stopping and removing old API container..."
	-docker-compose stop api
	-docker-compose rm -f api
	@echo "Removing old API image..."
	-docker rmi golang_api_server:latest
	@echo "Building new API image..."
	docker-compose build api
	@echo "Starting new API container..."
	docker-compose up -d api
	@echo "Waiting for API to be ready..."
	@sleep 10
	@echo "Cleaning up dangling images..."
	docker image prune -f
	@echo "✅ Deployment complete!"
	@echo "🚀 API: http://localhost:8080"
	@echo "🏥 Health check: http://localhost:8080/health"

reboot:
	@echo "🔄 Complete system reboot: stopping all services..."
	docker-compose down -v --remove-orphans
	@echo "🧹 Cleaning up old containers and volumes..."
	docker system prune -f
	@echo "🔨 Rebuilding API image..."
	docker-compose build api
	@echo "🚀 Starting all services fresh..."
	docker-compose up -d
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
build: docker-build
