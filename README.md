# Life Tracker Backend

Golang backend for tracking personal life data: users, activities, finances, notes, and time tracking.  
Built with clean architecture, JWT authentication, and observability support.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

---

## Features

- **Authentication**
  - JWT access & refresh tokens
  - Secure password hashing

- **User Management**
  - Create and update user profiles

- **Activity Tracking**
  - Register daily / weekly / monthly activities
  - Mark activities as done
  - Track completion history and total completion count

- **Finance Tracking**
  - Register fixed and variable expenses and incomes
  - Track spending and incomes records over time

- **Notes**
  - Write and store personal notes online
  - Simple note management per user

- **Time Tracking**
  - Track time spent per category (study, work, gym, etc.)
  - Aggregate total time per category

- **Monitoring & Observability**
  - Application metrics exposed for Grafana dashboards
  - Monitoring setup documented in the **infra repository**

- **Architecture & Quality**
  - Clean / DDD-inspired architecture
  - Modular domain separation
  - Input validation and security best practices
  - Integration tests for service layer across all domains

---

## Available Modules

Each module encapsulates its own business logic and data models internally.

- **auth** – Authentication & token handling  
- **user** – User profile management  
- **activity** – Habit and activity tracking  
- **finance** – Financial records and spending tracking  
- **note** – Personal notes management  
- **time** – Time tracking by category  

---

## Tech Stack

- **Go 1.21+**
- **Gin** – HTTP framework
- **GORM** – ORM
- **PostgreSQL** – Relational data
- **MongoDB** – Time-series / record-based data
- **JWT** – Authentication
- **Bcrypt** – Password hashing
- **Grafana** – Monitoring & metrics visualization

---

## Project Structure

```text
backend/
├── cmd/
│   └── main/
│       └── main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── domain/
│   │   ├── activity/
│   │   ├── auth/
│   │   ├── finance/
│   │   ├── note/
│   │   ├── time/
│   │   └── user/
│   ├── infrastructure/
│   └── middleware/
├── docs/
├── postman/
├── .env.example
├── .air.toml
├── Dockerfile
└── go.mod
```

---

## Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- Make

### Fastest Path (One Command)
```bash
git clone <your-repo-url>
cd backend
make start
```

Server runs at `http://localhost:8080`

> **Note:** This creates `.env` from `.env.example` and starts databases + app with hot reload.

### Standard Setup (More Control)
```bash
# 1. Clone and install dependencies
git clone <your-repo-url>
cd backend
go mod download

# 2. Setup environment and start
make start
```

### Manual Control
```bash
# Create environment file
make setup

# Start only databases
make db-up

# Run application directly
go run cmd/main/main.go

# Or with hot reload
make dev
```

### Available Commands

Run `make help` to see all available commands for development, testing, and database management.

---

## API Overview

Postman collections are available in the `postman/` folder.

- Auth
- Users
- Activities
- Finance
- Notes
- Time tracking
- Health & metrics

---

## Monitoring

This backend exposes metrics compatible with Grafana dashboards.  
Infrastructure and dashboards are defined in the **infra repository**.

---

## Development

### Prerequisites
Install required development tools:
```bash
make install-tools
```

### Running the Application
Start the application with development databases:
```bash
make dev
```

This will:
- Start PostgreSQL and Redis containers
- Wait for database readiness
- Launch the application with hot-reload via Air

### Testing

Tests are integration tests that run against real PostgreSQL and MongoDB instances.  
Each domain's service layer is fully covered.

| Domain    | Layer tested | Database       |
|-----------|-------------|----------------|
| auth      | service     | PostgreSQL     |
| user      | service     | PostgreSQL     |
| activity  | service     | PostgreSQL + MongoDB |
| finance   | service     | MongoDB        |
| note      | service     | PostgreSQL     |
| time      | service     | PostgreSQL     |
| screentime | integration | PostgreSQL + MongoDB |

**Prerequisites:**
- `.env.test` file (copy `.env.example` and point to test databases)
- Docker & Docker Compose (for managed test databases)
- `gotestsum` (`go install gotest.tools/gotestsum@latest`)

Run with managed test databases (starts containers, runs tests, stops containers):
```bash
make test
```

Run against already-running test databases:
```bash
make test-only
```

Tests run sequentially (`-p 1`) with the race detector enabled.

### Code Quality
Format and lint code:
```bash
make code-check
```

### Available Commands
Run `make help` to see all available commands for development, Docker, and database management.

---

## License

MIT License – free to use for learning and personal projects.
