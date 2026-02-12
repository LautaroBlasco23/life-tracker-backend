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
  - Tests included in each module

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
- PostgreSQL
- MongoDB
- Docker (optional)

### Installation

```bash
git clone <your-repo-url>
cd backend
go mod download
```

Create `.env` from `.env.example`.

```bash
go run cmd/main/main.go
```

Server runs at `http://localhost:8080`.

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

```bash
go install github.com/cosmtrek/air@latest
air
```

```bash
go test ./...
```

---

## License

MIT License – free to use for learning and personal projects.
