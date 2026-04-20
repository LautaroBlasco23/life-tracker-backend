package repository

import (
	"fmt"
	"os"
	"testing"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/auth/model"
	userModel "life-tracker-backend/internal/domain/user/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB *gorm.DB
	cfg    *config.Config
)

// Subject: internal/domain/auth/repository - AuthRepository and UserRepository against real Postgres.
// Scope:   SQL correctness for authentication-related database operations.
// Out of scope:
//   - service-layer business rules    → auth/service tests (unit)
//   - HTTP contract                   → auth/controller tests
// Infrastructure: PostgreSQL test database (started via make test or docker-compose).
// Data strategy: truncate tables after each test.
// Parallel-safe: no - tests run sequentially with shared testDB.

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	code := m.Run()
	os.Exit(code)
}

func setupTestDatabase(t *testing.T) {
	if testDB == nil {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		require.NoError(t, err, "Failed to connect to PostgreSQL")

		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.SetMaxOpenConns(1)

		// Migrate both Auth and User models
		require.NoError(t, db.AutoMigrate(&userModel.User{}))
		require.NoError(t, db.AutoMigrate(&model.Auth{}))
		testDB = db
	}
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	// Clean both auth and users tables
	require.NoError(t, testDB.Exec("TRUNCATE TABLE auths RESTART IDENTITY CASCADE;").Error)
	require.NoError(t, testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE;").Error)
}

func getAuthRepository(t *testing.T) AuthRepository {
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewAuthRepository(testDB)
}

func getUserRepository(t *testing.T) UserRepository {
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewUserRepository(testDB)
}

// createTestUser creates a user in the database and returns its ID.
// Used for tests that need a valid user reference.
func createTestUser(t *testing.T, firstName, lastName string) uint {
	user := &userModel.User{
		FirstName: firstName,
		LastName:  lastName,
	}
	err := testDB.Create(user).Error
	require.NoError(t, err)
	return user.ID
}

// createTestAuth creates an auth record in the database.
// Used for tests that need an existing auth record.
func createTestAuth(t *testing.T, userID uint, email, passwordHash string) *model.Auth {
	auth := &model.Auth{
		UserID:       userID,
		Email:        email,
		PasswordHash: passwordHash,
		Username:     fmt.Sprintf("user%d", userID),
	}
	err := testDB.Create(auth).Error
	require.NoError(t, err)
	return auth
}

// ============================================================================
// AuthRepository Tests
// ============================================================================

func TestAuthRepository_Create(t *testing.T) {
	authRepo := getAuthRepository(t)

	t.Run("create auth with valid data", func(t *testing.T) {
		// First create a user since UserID is required
		userID := createTestUser(t, "Test", "User")

		auth := &model.Auth{
			UserID:       userID,
			Email:        "test@example.com",
			PasswordHash: "hashedpassword123",
		}

		err := authRepo.Create(auth)

		assert.NoError(t, err)
		assert.NotZero(t, auth.ID)
		assert.NotZero(t, auth.CreatedAt)
		assert.NotZero(t, auth.UpdatedAt)
	})

	t.Run("create auth with duplicate email fails", func(t *testing.T) {
		// Create first user and auth
		userID1 := createTestUser(t, "Test", "User1")
		createTestAuth(t, userID1, "duplicate@example.com", "password123")

		// Create second user
		userID2 := createTestUser(t, "Test", "User2")

		// Try to create auth with duplicate email but different username
		auth := &model.Auth{
			UserID:       userID2,
			Email:        "duplicate@example.com",
			PasswordHash: "password456",
			Username:     fmt.Sprintf("user%d", userID2),
		}

		err := authRepo.Create(auth)

		assert.Error(t, err)
		// Should fail due to unique constraint violation on email
	})
}

func TestAuthRepository_FindByEmail(t *testing.T) {
	authRepo := getAuthRepository(t)

	t.Run("find auth by existing email", func(t *testing.T) {
		// Setup: create user and auth
		userID := createTestUser(t, "Test", "User")
		createTestAuth(t, userID, "findme@example.com", "password123")

		found, err := authRepo.FindByEmail("findme@example.com")

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "findme@example.com", found.Email)
		assert.Equal(t, userID, found.UserID)
		assert.Equal(t, "password123", found.PasswordHash)
	})

	t.Run("find auth by non-existent email returns not found", func(t *testing.T) {
		found, err := authRepo.FindByEmail("nonexistent@example.com")

		assert.Error(t, err)
		assert.Nil(t, found)
		assert.ErrorIs(t, err, ErrAuthNotFound)
	})
}

func TestAuthRepository_FindByUserID(t *testing.T) {
	authRepo := getAuthRepository(t)

	t.Run("find auth by existing user ID", func(t *testing.T) {
		// Setup: create user and auth
		userID := createTestUser(t, "Test", "User")
		createTestAuth(t, userID, "byuserid@example.com", "password123")

		found, err := authRepo.FindByUserID(userID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, userID, found.UserID)
		assert.Equal(t, "byuserid@example.com", found.Email)
	})

	t.Run("find auth by non-existent user ID returns not found", func(t *testing.T) {
		found, err := authRepo.FindByUserID(9999)

		assert.Error(t, err)
		assert.Nil(t, found)
		assert.ErrorIs(t, err, ErrAuthNotFound)
	})
}

func TestAuthRepository_EmailExists(t *testing.T) {
	authRepo := getAuthRepository(t)

	t.Run("email exists returns true", func(t *testing.T) {
		// Setup: create user and auth
		userID := createTestUser(t, "Test", "User")
		createTestAuth(t, userID, "exists@example.com", "password123")

		exists, err := authRepo.EmailExists("exists@example.com")

		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("email does not exist returns false", func(t *testing.T) {
		exists, err := authRepo.EmailExists("doesnotexist@example.com")

		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestAuthRepository_UpdatePassword(t *testing.T) {
	authRepo := getAuthRepository(t)

	t.Run("update password for existing user", func(t *testing.T) {
		// Setup: create user and auth
		userID := createTestUser(t, "Test", "User")
		createTestAuth(t, userID, "updatepass@example.com", "oldpassword")

		err := authRepo.UpdatePassword(userID, "newpasswordhash")

		assert.NoError(t, err)

		// Verify the password was updated
		updated, err := authRepo.FindByUserID(userID)
		require.NoError(t, err)
		assert.Equal(t, "newpasswordhash", updated.PasswordHash)
	})

	t.Run("update password for non-existent user returns not found", func(t *testing.T) {
		err := authRepo.UpdatePassword(9999, "newpasswordhash")

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthNotFound)
	})
}

func TestAuthRepository_UpdateEmail(t *testing.T) {
	authRepo := getAuthRepository(t)

	t.Run("update email for existing user", func(t *testing.T) {
		// Setup: create user and auth
		userID := createTestUser(t, "Test", "User")
		createTestAuth(t, userID, "oldemail@example.com", "password123")

		err := authRepo.UpdateEmail(userID, "newemail@example.com")

		assert.NoError(t, err)

		// Verify the email was updated
		updated, err := authRepo.FindByUserID(userID)
		require.NoError(t, err)
		assert.Equal(t, "newemail@example.com", updated.Email)

		// Verify we can no longer find by old email
		_, err = authRepo.FindByEmail("oldemail@example.com")
		assert.ErrorIs(t, err, ErrAuthNotFound)
	})

	t.Run("update email for non-existent user returns not found", func(t *testing.T) {
		err := authRepo.UpdateEmail(9999, "newemail@example.com")

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthNotFound)
	})
}

// ============================================================================
// UserRepository Tests
// ============================================================================

func TestUserRepository_Create(t *testing.T) {
	userRepo := getUserRepository(t)

	t.Run("create user with valid data", func(t *testing.T) {
		user := &userModel.User{
			FirstName: "John",
			LastName:  "Doe",
		}

		err := userRepo.Create(user)

		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
		assert.NotZero(t, user.CreatedAt)
		assert.NotZero(t, user.UpdatedAt)
	})

	t.Run("create user without first name fails", func(t *testing.T) {
		user := &userModel.User{
			FirstName: "",
			LastName:  "Doe",
		}

		err := userRepo.Create(user)

		assert.Error(t, err)
	})

	t.Run("create user without last name fails", func(t *testing.T) {
		user := &userModel.User{
			FirstName: "John",
			LastName:  "",
		}

		err := userRepo.Create(user)

		assert.Error(t, err)
	})


}

func TestUserRepository_FindByID(t *testing.T) {
	userRepo := getUserRepository(t)

	t.Run("find user by existing ID", func(t *testing.T) {
		// Setup: create user
		userID := createTestUser(t, "Jane", "Smith")

		found, err := userRepo.FindByID(userID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, userID, found.ID)
		assert.Equal(t, "Jane", found.FirstName)
		assert.Equal(t, "Smith", found.LastName)
	})

	t.Run("find user by non-existent ID returns not found", func(t *testing.T) {
		found, err := userRepo.FindByID(9999)

		assert.Error(t, err)
		assert.Nil(t, found)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}
