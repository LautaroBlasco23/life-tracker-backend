# Golang REST API with Clean Architecture

A production-ready REST API built with Go, featuring JWT authentication, clean architecture, and comprehensive CRUD operations.

## 🏗️ Project Structure

```
project/
├── cmd/
│   └── main/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── model/
│   │   │   └── auth.go
│   │   ├── service/
│   │   │   └── auth_service.go
│   │   ├── controller/
│   │   │   └── auth_controller.go
│   │   └── routes/
│   │       └── auth_routes.go
│   ├── user/
│   │   ├── model/
│   │   │   └── user.go
│   │   ├── service/
│   │   │   └── user_service.go
│   │   ├── controller/
│   │   │   └── user_controller.go
│   │   └── routes/
│   │       └── user_routes.go
│   ├── middleware/
│   │   └── auth_middleware.go
│   ├── config/
│   │   └── config.go
│   └── database/
│       └── database.go
├── .env
├── go.mod
└── README.md
```

## 🚀 Features

- **Clean Architecture**: Organized by feature modules with clear separation of concerns
- **JWT Authentication**: Secure token-based authentication with refresh tokens
- **Password Hashing**: Bcrypt for secure password storage
- **Environment Configuration**: Flexible configuration via environment variables
- **Database Integration**: GORM with PostgreSQL support
- **Middleware**: Authentication and CORS middleware
- **Input Validation**: Request validation using Gin binding
- **Error Handling**: Comprehensive error handling and responses
- **RESTful API**: Following REST conventions
- **Soft Deletes**: GORM soft delete support
- **Auto Migration**: Automatic database schema migration

## 🛠️ Tech Stack

- **Go 1.21+**
- **Gin** - HTTP web framework
- **GORM** - ORM for database operations
- **PostgreSQL** - Primary database
- **JWT** - Authentication tokens
- **Bcrypt** - Password hashing
- **Godotenv** - Environment variable management

## 📋 Prerequisites

- Go 1.21 or higher
- PostgreSQL database
- Git

## ⚡ Quick Start

### 1. Clone the repository
```bash
git clone <your-repo-url>
cd golang-rest-api
```

### 2. Install dependencies
```bash
go mod download
```

### 3. Setup environment variables
Create a `.env` file in the root directory (see `.env` example above)

### 4. Setup PostgreSQL database
Create a PostgreSQL database with the name specified in your `.env` file.

### 5. Run the application
```bash
go run cmd/main/main.go
```

The server will start on the port specified in your `.env` file (default: 8080).

## 📚 API Endpoints

### Authentication Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/register` | Register a new user |
| POST | `/api/v1/auth/login` | Login user |
| POST | `/api/v1/auth/refresh` | Refresh access token |

### User Endpoints (Protected)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users/profile` | Get current user profile |
| PUT | `/api/v1/users/profile` | Update current user profile |
| GET | `/api/v1/users` | Get all users |
| GET | `/api/v1/users/:id` | Get user by ID |
| DELETE | `/api/v1/users/:id` | Delete user by ID |

### Health Check
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check endpoint |

## 📖 API Usage Examples

### Register a new user
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@example.com",
    "password": "password123",
    "firstName": "John",
    "lastName": "Doe"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@example.com",
    "password": "password123"
  }'
```

### Get user profile
```bash
curl -X GET http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Update user profile
```bash
curl -X PUT http://localhost:8080/api/v1/users/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "firstName": "Jonathan",
    "profilePicUrl": "https://example.com/profile.jpg"
  }'
```

### Refresh token
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "YOUR_REFRESH_TOKEN"
  }'
```

## 🔧 Development

### Hot Reload with Air
1. Install Air:
   ```bash
   go install github.com/cosmtrek/air@latest
   ```

2. Run with hot reload:
   ```bash
   air
   ```

### Running Tests
```bash
go test ./...
```

## 🏗️ Architecture Benefits

### Separation of Concerns
- **Models**: Data structures and database schemas
- **Services**: Business logic and data processing
- **Controllers**: HTTP request/response handling
- **Routes**: API route definitions
- **Middleware**: Cross-cutting concerns (auth, CORS, logging)

### Testability
Each layer can be tested independently:
- Unit tests for services (business logic)
- Integration tests for controllers
- End-to-end tests for routes

### Scalability
- Easy to add new features
- Clear boundaries between components
- Easy to maintain and refactor

## 🔒 Security Features

- **Password Hashing**: Using bcrypt with salt
- **JWT Tokens**: Secure token-based authentication
- **Token Expiry**: Configurable token expiration
- **Refresh Tokens**: Secure token refresh mechanism
- **Input Validation**: Request validation and sanitization
- **CORS Protection**: Cross-Origin Resource Sharing configuration

## 📊 Database Schema

### Users Table
- `id` (Primary Key)
- `first_name` (Required)
- `last_name` (Required)
- `email` (Unique, Required)
- `profile_pic_url` (Optional)
- `created_at`
- `updated_at`
- `deleted_at` (Soft Delete)

### Auth Table
- `id` (Primary Key)
- `email` (Unique, Required)
- `password_hash` (Required)
- `user_id` (Foreign Key to Users)
- `created_at`
- `updated_at`
- `deleted_at` (Soft Delete)

## 🚀 Production Recommendations

### Security
- Use strong JWT secrets
- Implement rate limiting
- Add request logging
- Use HTTPS in production
- Validate and sanitize all inputs

### Performance
- Add database indexing
- Implement caching (Redis)
- Use connection pooling
- Add monitoring and metrics

### Deployment
- Use Docker containers
- Set up CI/CD pipelines
- Configure environment-specific settings
- Add health checks and monitoring

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Gin Web Framework](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [JWT-Go](https://github.com/golang-jwt/jwt)
- [Godotenv](https://github.com/joho/godotenv)
