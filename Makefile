# Golang REST API Makefile
.PHONY: dev dev-watch db-up db-down db-reset mongo-up mongo-down mongo-reset db-all-up db-all-down db-all-reset help

# Default target
help:
	@echo "Available commands:"
	@echo "  dev             - Run the Go application in development mode"
	@echo "  dev-watch       - Run with hot reload (restarts on file changes)"
	@echo ""
	@echo "Database Commands:"
	@echo "  db-up           - Start PostgreSQL database container"
	@echo "  db-down         - Stop PostgreSQL database container"
	@echo "  db-reset        - Reset PostgreSQL database (stop, remove, start fresh)"
	@echo ""
	@echo "MongoDB Commands:"
	@echo "  mongo-up        - Start MongoDB and Mongo Express containers"
	@echo "  mongo-down      - Stop MongoDB and Mongo Express containers"
	@echo "  mongo-reset     - Reset MongoDB database (stop, remove, start fresh)"
	@echo ""
	@echo "All Databases Commands:"
	@echo "  db-all-up       - Start all databases (PostgreSQL + MongoDB)"
	@echo "  db-all-down     - Stop all databases"
	@echo "  db-all-reset    - Reset all databases"

# Run the application in development mode
dev:
	@echo "Starting Go application..."
	go run cmd/main/main.go

# Run with hot reload
dev-watch:
	@echo "Starting Go application with hot reload..."
	@if ! command -v air > /dev/null 2>&1; then \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
	fi
	$(shell go env GOPATH)/bin/air -c .air.toml

# Database commands (manages both PostgreSQL and MongoDB)
db-up:
	@echo "Starting all databases..."
	docker-compose up -d postgres mongodb
	@echo "Waiting for databases to be ready..."
	@sleep 10
	@echo "✅ All databases are ready!"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 MongoDB: localhost:27017"
	@echo "🌐 Mongo Express: http://localhost:8081 (admin/admin123)"

db-down:
	@echo "Stopping all databases..."
	docker-compose down
	@echo "✅ All databases stopped!"

db-reset:
	@echo "Resetting all databases..."
	@echo "Stopping and removing all containers..."
	docker-compose down
	@echo "Removing all volumes..."
	docker-compose down -v
	@echo "Starting fresh databases..."
	docker-compose up -d postgres mongodb
	@echo "Waiting for databases to be ready..."
	@sleep 15
	@echo "✅ All databases reset complete!"
	@echo "📊 PostgreSQL: localhost:5432"
	@echo "📊 MongoDB: localhost:27017"
