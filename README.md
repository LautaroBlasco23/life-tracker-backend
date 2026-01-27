# Life Tracker Backend

Golang application for life tracking with JWT authentication, user management, and activity tracking featuring completion records and statistics.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

## Git Hooks Setup

This project uses versioned git hooks.

Run once after cloning:

```bash
git config core.hooksPath .githooks
chmod +x .githooks/*
```

## Features

- **Authentication**: JWT tokens with refresh mechanism
- **User Management**: Complete CRUD operations for users
- **Activity Tracking**: Create and track daily, weekly, monthly, or one-time activities
- **Activity Records**: Store completion history with MongoDB
- **Activity Stats**: Track streaks, completion rates, and history
- **Hybrid Database**: PostgreSQL for relational data, MongoDB for time-series records
- **Clean Architecture**: Separated concerns with clear domain boundaries
- **Security**: Bcrypt password hashing, JWT tokens, input validation
- **Soft Deletes**: GORM soft delete support

## Tech Stack

- **Go 1.21+** - Programming language
- **Gin** - HTTP web framework
- **GORM** - ORM for database operations
- **PostgreSQL** - Relational database
- **MongoDB** - Document database for activity records
- **JWT** - Token-based authentication
- **Bcrypt** - Password hashing

## Project Structure

```

project/
├── cmd/main/              # Application entry point
├── internal/
│   ├── auth/              # Authentication module
│   ├── user/              # User module
│   ├── domain/activity/   # Activity domain
│   ├── middleware/        # HTTP middleware
│   ├── config/            # Configuration
│   └── database/          # Database setup
├── .env
└── go.mod

```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL
- MongoDB
- Docker (for testing)

### Installation

1. **Clone and install dependencies**

```bash
git clone <your-repo-url>
cd golang-rest-api
go mod download
```

1. **Setup environment variables**

Create a `.env` file:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=your_database
DB_SSLMODE=disable

MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=life_tracker

JWT_SECRET=your-secret-key-here
JWT_EXPIRY=15m
REFRESH_TOKEN_EXPIRY=7d

SERVER_PORT=8080
```

1. **Create databases**

```bash
createdb your_database
```

1. **Run the application**

```bash
go run cmd/main/main.go
```

Server starts at `http://localhost:8080`

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Login user |
| POST | `/api/v1/auth/refresh` | Refresh access token |

### Users (Protected)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users/profile` | Get current user |
| PUT | `/api/v1/users/profile` | Update current user |
| GET | `/api/v1/users` | List all users |
| GET | `/api/v1/users/:id` | Get user by ID |
| DELETE | `/api/v1/users/:id` | Delete user |

### Activities (Protected)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/activities` | Create activity |
| GET | `/api/v1/activities` | List user activities |
| GET | `/api/v1/activities/:id` | Get activity by ID |
| PUT | `/api/v1/activities/:id` | Update activity |
| DELETE | `/api/v1/activities/:id` | Delete activity |
| POST | `/api/v1/activities/:id/complete` | Mark activity complete |
| GET | `/api/v1/activities/:id/stats` | Get activity statistics |

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |

## Usage Examples

**Register a user**

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "firstName": "John",
    "lastName": "Doe"
  }'
```

**Login**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

**Get profile**

```bash
curl -X GET http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

**Create activity**

```bash
curl -X POST http://localhost:8080/api/v1/activities \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Morning Exercise",
    "description": "30 minutes workout",
    "frequency": "daily",
    "target": 30
  }'
```

## Database Schema

### PostgreSQL

**Users**

```
id              UUID (PK)
first_name      VARCHAR (required)
last_name       VARCHAR (required)
email           VARCHAR (unique, required)
profile_pic_url VARCHAR (optional)
created_at      TIMESTAMP
updated_at      TIMESTAMP
deleted_at      TIMESTAMP (soft delete)
```

**Auth**

```
id            UUID (PK)
email         VARCHAR (unique, required)
password_hash VARCHAR (required)
user_id       UUID (FK -> users.id)
created_at    TIMESTAMP
updated_at    TIMESTAMP
deleted_at    TIMESTAMP (soft delete)
```

**Activities**

```
id          UUID (PK)
user_id     UUID (FK -> users.id)
name        VARCHAR (required)
description TEXT
frequency   VARCHAR (daily/weekly/monthly/one_time)
target      INTEGER
created_at  TIMESTAMP
updated_at  TIMESTAMP
deleted_at  TIMESTAMP (soft delete)
```

### MongoDB

**Activity Records**

```
{
  activity_id: UUID,
  user_id: UUID,
  completed_at: ISODate,
  value: Number,
  notes: String
}
```

## Testing

### Quick Start

```bash
make test
make test-coverage
```

### Available Commands

```bash
make test              # Run all tests
make test-unit         # Run unit tests only  
make test-integration  # Run integration tests only
make test-coverage     # Generate HTML coverage report
make test-bench        # Run benchmark tests
make clean-test        # Stop containers and clean artifacts
```

### Requirements

- Docker installed and running
- Go 1.21+
- Make

### Test Structure

```
internal/domain/activity/
├── service/
│   ├── activity_service.go
│   └── activity_service_test.go
└── controller/
    ├── activity_controller.go
    └── activity_controller_test.go
```

### Test Databases

Tests use Docker containers automatically managed by the Makefile:

- **PostgreSQL**: `localhost:5432` (user: testuser, pass: testpass)
- **MongoDB**: `localhost:27017`

Containers are created before tests and cleaned up afterward.

### CI/CD Integration

**GitHub Actions**

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run tests
        run: make test
```

**GitLab CI**

```yaml
test:
  image: golang:1.21
  script:
    - make test
```

### Troubleshooting

**"Docker daemon not running"**

Start Docker Desktop or Docker service:

```bash
# Linux
sudo systemctl start docker
# macOS
open -a Docker
```

**"Port already in use"**

Stop existing containers:

```bash
make clean-test
```

**Tests hanging**

Check Docker logs:

```bash
docker logs life-tracker-test-mongo
docker logs life-tracker-test-postgres
```

## Development

**Hot reload with Air**

```bash
go install github.com/cosmtrek/air@latest
air
```

**Run tests**

```bash
go test ./...
```

## Production Considerations

- Use strong JWT secrets
- Enable HTTPS
- Add rate limiting for public APIs
- Add database indexes on email and frequently queried fields
- Implement logging and monitoring
- Set up database backups
- Use connection pooling for both PostgreSQL and MongoDB

## License

MIT License - feel free to use this project for learning and personal projects.
