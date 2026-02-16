package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/auth/dto"
	"life-tracker-backend/internal/domain/auth/model"
	"life-tracker-backend/internal/domain/auth/service"
	userModel "life-tracker-backend/internal/domain/user/model"
	userService "life-tracker-backend/internal/domain/user/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var cfg *config.Config

func init() {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(t *testing.T) (*gin.Engine, *service.AuthService, func()) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to PostgreSQL")

	db.Exec("DROP TABLE IF EXISTS auths CASCADE")
	db.Exec("DROP TABLE IF EXISTS users CASCADE")
	require.NoError(t, db.AutoMigrate(&model.Auth{}, &userModel.User{}))

	authService := service.NewAuthService(db, cfg)
	userSvc := userService.NewUserService(db, nil)
	controller := NewAuthController(authService, userSvc)

	router := gin.New()

	auth := router.Group("/auth")
	{
		auth.POST("/register", controller.Register)
		auth.POST("/login", controller.Login)
		auth.POST("/refresh", controller.RefreshToken)

		authenticated := auth.Group("")
		authenticated.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		{
			authenticated.PUT("/password", controller.UpdatePassword)
			authenticated.PUT("/email", controller.UpdateEmail)
		}
	}

	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS auths CASCADE")
		db.Exec("DROP TABLE IF EXISTS users CASCADE")
	}

	return router, authService, cleanup
}

func TestRegisterHandler(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("successful registration", func(t *testing.T) {
		req := dto.RegisterRequest{
			Email:     "test@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "User registered successfully", resp["message"])
		assert.NotNil(t, resp["data"])
	})

	t.Run("duplicate email", func(t *testing.T) {
		req := dto.RegisterRequest{
			Email:     "duplicate@example.com",
			Password:  "password123",
			FirstName: "Jane",
			LastName:  "Doe",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)
		require.Equal(t, http.StatusCreated, w.Code)

		w = httptest.NewRecorder()
		httpReq = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		body := []byte(`{"invalid": "data"}`)

		httpReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestLoginHandler(t *testing.T) {
	router, authService, cleanup := setupTestRouter(t)
	defer cleanup()

	registerReq := &dto.RegisterRequest{
		Email:     "login@example.com",
		Password:  "password123",
		FirstName: "Login",
		LastName:  "Test",
	}
	_, _, err := authService.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful login", func(t *testing.T) {
		req := dto.LoginRequest{
			Email:    "login@example.com",
			Password: "password123",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Login successful", resp["message"])
		assert.NotNil(t, resp["data"])
	})

	t.Run("invalid credentials", func(t *testing.T) {
		req := dto.LoginRequest{
			Email:    "login@example.com",
			Password: "wrongpassword",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRefreshTokenHandler(t *testing.T) {
	router, authService, cleanup := setupTestRouter(t)
	defer cleanup()

	registerReq := &dto.RegisterRequest{
		Email:     "refresh@example.com",
		Password:  "password123",
		FirstName: "Refresh",
		LastName:  "Test",
	}
	tokens, _, err := authService.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful token refresh", func(t *testing.T) {
		req := dto.RefreshRequest{
			RefreshToken: tokens.RefreshToken,
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Token refreshed successfully", resp["message"])
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		req := dto.RefreshRequest{
			RefreshToken: "invalid-token",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestUpdatePasswordHandler(t *testing.T) {
	router, authService, cleanup := setupTestRouter(t)
	defer cleanup()

	registerReq := &dto.RegisterRequest{
		Email:     "password@example.com",
		Password:  "oldpassword",
		FirstName: "Password",
		LastName:  "Test",
	}
	_, _, err := authService.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful password update", func(t *testing.T) {
		req := dto.UpdatePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPut, "/auth/password", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Password updated successfully", resp["message"])
	})

	t.Run("incorrect current password", func(t *testing.T) {
		req := dto.UpdatePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword123",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPut, "/auth/password", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUpdateEmailHandler(t *testing.T) {
	router, authService, cleanup := setupTestRouter(t)
	defer cleanup()

	registerReq := &dto.RegisterRequest{
		Email:     "email@example.com",
		Password:  "password123",
		FirstName: "Email",
		LastName:  "Test",
	}
	_, _, err := authService.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful email update", func(t *testing.T) {
		req := dto.UpdateEmailRequest{
			Password: "password123",
			NewEmail: "newemail@example.com",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPut, "/auth/email", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Email updated successfully", resp["message"])
		assert.NotNil(t, resp["data"])
	})

	t.Run("email already in use", func(t *testing.T) {
		otherReq := &dto.RegisterRequest{
			Email:     "existing@example.com",
			Password:  "password123",
			FirstName: "Other",
			LastName:  "User",
		}
		_, _, err := authService.Register(otherReq)
		require.NoError(t, err)

		req := dto.UpdateEmailRequest{
			Password: "password123",
			NewEmail: "existing@example.com",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPut, "/auth/email", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUnauthorizedAccess(t *testing.T) {
	router := gin.New()
	controller := NewAuthController(nil, nil)

	auth := router.Group("/auth")
	{
		auth.PUT("/password", controller.UpdatePassword)
	}

	t.Run("missing user context", func(t *testing.T) {
		req := dto.UpdatePasswordRequest{
			CurrentPassword: "old",
			NewPassword:     "new",
		}
		body, _ := json.Marshal(req)

		httpReq := httptest.NewRequest(http.MethodPut, "/auth/password", bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
