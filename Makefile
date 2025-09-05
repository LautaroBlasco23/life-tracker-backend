# Golang REST API Makefile

.PHONY: dev db-up db-down db-reset help

# Default target
help:
	@echo "Available commands:"
	@echo "  dev      - Run the Go application in development mode"
	@echo "  db-up    - Start PostgreSQL database container"
	@echo "  db-down  - Stop PostgreSQL database container"
	@echo "  db-reset - Reset database (stop, remove container and volumes, then start fresh)"

# Run the application in development mode
dev:
	@echo "Starting Go application..."
	go run cmd/main/main.go

# Start the database container
db-up:
	@echo "Starting PostgreSQL database..."
	docker-compose up -d postgres
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Database is ready!"

# Stop the database container
db-down:
	@echo "Stopping PostgreSQL database..."
	docker-compose down

# Reset database - remove everything and start fresh
db-reset:
	@echo "Resetting PostgreSQL database..."
	@echo "Stopping and removing containers..."
	docker-compose down
	@echo "Removing volumes..."
	docker-compose down -v
	@echo "Removing container..."
	docker rm -f golang_api_postgres 2>/dev/null || true
	@echo "Starting fresh database..."
	docker-compose up -d postgres
	@echo "Waiting for database to be ready..."
	@sleep 10
	@echo "Database reset complete!"
