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
	"life-tracker-backend/internal/domain/user/dto"
	"life-tracker-backend/internal/domain/user/model"
	"life-tracker-backend/internal/domain/user/service"

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

func setupTestRouter(t *testing.T) (*gin.Engine, *service.UserService, func()) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to PostgreSQL")

	db.Exec("DROP TABLE IF EXISTS users CASCADE")
	require.NoError(t, db.AutoMigrate(&model.User{}))

	userService := service.NewUserService(db, nil)
	controller := NewUserController(userService)

	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("email", "test@example.com")
		c.Next()
	})

	users := router.Group("/users")
	{
		users.GET("/me", controller.GetMyProfile)
		users.PUT("/me", controller.UpdateProfile)
		users.GET("", controller.GetAllUsers)
		users.GET("/:id", controller.GetUserByID)
		users.DELETE("/:id", controller.DeleteUser)
	}

	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS users CASCADE")
	}

	return router, userService, cleanup
}

func createTestUser(t *testing.T, db *gorm.DB, id uint) {
	user := &model.User{
		ID:        id,
		FirstName: "Test",
		LastName:  "User",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
}

func TestGetMyProfileHandler(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("get my profile successfully", func(t *testing.T) {
		db, _ := gorm.Open(postgres.Open(fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)), &gorm.Config{})
		createTestUser(t, db, 1)

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Profile retrieved successfully", resp["message"])

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "Test", data["firstName"])
		assert.Equal(t, "test@example.com", data["email"])
	})

	t.Run("user not found", func(t *testing.T) {
		customRouter := gin.New()
		customRouter.Use(func(c *gin.Context) {
			c.Set("userID", uint(9999))
			c.Set("email", "notfound@example.com")
			c.Next()
		})

		db, _ := gorm.Open(postgres.Open(fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)), &gorm.Config{})
		userService := service.NewUserService(db, nil)
		controller := NewUserController(userService)

		customRouter.GET("/users/me", controller.GetMyProfile)

		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		customRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateProfileHandler(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("update profile successfully", func(t *testing.T) {
		db, _ := gorm.Open(postgres.Open(fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)), &gorm.Config{})
		createTestUser(t, db, 1)

		newFirstName := "Updated"
		updateReq := dto.UpdateUserRequest{
			FirstName: &newFirstName,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "Updated", data["firstName"])
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetAllUsersHandler(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("get all users", func(t *testing.T) {
		db, _ := gorm.Open(postgres.Open(fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)), &gorm.Config{})

		for i := 1; i <= 3; i++ {
			createTestUser(t, db, uint(i))
		}

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(3), resp["count"])
	})
}

func TestGetUserByIDHandler(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("get user by id successfully", func(t *testing.T) {
		db, _ := gorm.Open(postgres.Open(fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)), &gorm.Config{})
		createTestUser(t, db, 1)

		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(1), data["id"])
	})

	t.Run("invalid user id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/9999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDeleteUserHandler(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Run("delete user successfully", func(t *testing.T) {
		db, _ := gorm.Open(postgres.Open(fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)), &gorm.Config{})
		createTestUser(t, db, 2)

		req := httptest.NewRequest(http.MethodDelete, "/users/2", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "User deleted successfully", resp["message"])
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/users/9999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUnauthorizedAccess(t *testing.T) {
	router := gin.New()
	controller := NewUserController(nil)

	router.GET("/users/me", controller.GetMyProfile)

	t.Run("missing user context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
