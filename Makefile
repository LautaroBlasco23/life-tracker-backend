.PHONY: help setup start install-tools code-check dev docker-up docker-down docker-build db-up db-down db-remove db-test-up db-test-down db-test-remove test test-only hooks

.DEFAULT_GOAL := help

help:
	@echo ""
	@echo "  🚀 Quick Start:"
	@echo "    setup              - Create .env from .env.example"
	@echo "    start              - Setup environment and start application (setup + dev)"
	@echo ""
	@echo "  🛠️  Development:"
	@echo "    install-tools      - Install Go tools (gofumpt, golangci-lint, air, gotestsum)"
	@echo "    code-check         - Format and lint code"
	@echo "    dev                - Start application with databases"
	@echo "    test               - Run all tests with test database"
	@echo "    test-only          - Run tests without starting database"
	@echo ""
	@echo "  🐳 Docker:"
	@echo "    docker-up          - Start all services"
	@echo "    docker-down        - Stop services"
	@echo "    docker-build       - Build API image"
	@echo ""
	@echo "  🗄️  Database (Development):"
	@echo "    db-up              - Start development databases"
	@echo "    db-down            - Stop development databases"
	@echo "    db-remove          - Remove development databases and volumes"
	@echo ""
	@echo "  🧪 Database (Test):"
	@echo "    db-test-up         - Start test databases"
	@echo "    db-test-down       - Stop test databases"
	@echo "    db-test-remove     - Remove test databases and volumes"

setup:
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
		echo "✓ .env created. Review and update configuration if needed."; \
	else \
		echo "✓ .env already exists"; \
	fi

start: setup dev

install-tools:
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2
	go install github.com/air-verse/air@latest
	go install gotest.tools/gotestsum@latest

code-check: 
	gofumpt -l -w .
	golangci-lint run --fix ./...

dev: hooks db-up
	@until docker exec golang_api_postgres pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
	ENV_FILE=.env air -c .air.toml

docker-up:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker compose --env-file .env up -d

docker-down:
	docker compose down

docker-build:
	@[ -f .env ] || (echo ".env not found"; exit 1)
	docker compose build api

db-up:
	docker compose -f docker-compose.db.yml --env-file .env up -d

db-down:
	docker compose -f docker-compose.db.yml --env-file .env stop

db-remove:
	docker compose -f docker-compose.db.yml --env-file .env down -v

db-test-up:
	@[ -f .env.test ] || (echo ".env.test not found"; exit 1)
	docker compose -f docker-compose.db.yml --env-file .env.test up -d

db-test-down:
	@[ -f .env.test ] || (echo ".env.test not found"; exit 1)
	docker compose -f docker-compose.db.yml --env-file .env.test stop

db-test-remove:
	@[ -f .env.test ] || (echo ".env.test not found"; exit 1)
	docker compose -f docker-compose.db.yml --env-file .env.test down -v

test: db-test-up
	@ENVIRONMENT=test gotestsum --format=short-verbose -- -race -p 1 ./...; \
	TEST_EXIT=$$?; \
	$(MAKE) db-test-down; \
	exit $$TEST_EXIT

test-only:
	ENVIRONMENT=test gotestsum --format=short-verbose -- -race -p 1 ./...

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/*
