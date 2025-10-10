# Life Tracker Backend

Golang App developed to handle life tracker's app logic, featuring JWT authentication, user management, and activity tracking with completion records.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

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
- **PostgreSQL** - Database
- **JWT** - Token-based authentication
- **Bcrypt** - Password hashing

## Project Structure

```
project/
├── cmd/main/              # Application entry point
├── internal/
│   ├── auth/              # Authentication module
│   ├── user/              # User module
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

### Installation

1. **Clone and install dependencies**
```bash
git clone <your-repo-url>
cd golang-rest-api
go mod download
```

2. **Setup environment variables**

Create a `.env` file:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=your_database
DB_SSLMODE=disable

JWT_SECRET=your-secret-key-here
JWT_EXPIRY=15m
REFRESH_TOKEN_EXPIRY=7d

SERVER_PORT=8080
```

3. **Create PostgreSQL database**
```bash
createdb your_database
```

4. **Run the application**
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

**Get profile (authenticated)**
```bash
curl -X GET http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Database Schema

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

## Notes

- Use strong JWT secrets in production
- Enable HTTPS for production deployments
- Consider adding rate limiting for public APIs
- Database indexes are recommended for email fields
- Implement logging and monitoring for production use

## License

MIT License - feel free to use this project for learning and personal projects.
