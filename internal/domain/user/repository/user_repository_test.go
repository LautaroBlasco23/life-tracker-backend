package repository

import (
	"fmt"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/user/model"

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

// TestMain initializes the test environment by loading configuration
// and setting the environment to "test" before running the test suite.
func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	code := m.Run()
	os.Exit(code)
}

// setupTestDatabase establishes a connection to the PostgreSQL test database.
// It configures a single connection pool, applies AutoMigrate for the User model,
// and caches the connection for reuse across tests.
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

		require.NoError(t, db.AutoMigrate(&model.User{}))
		testDB = db
	}
}

// cleanDatabase truncates the users table and resets the identity sequence
// to ensure each test starts with a clean database state.
func cleanDatabase(t *testing.T) {
	t.Helper()
	require.NoError(t, testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE;").Error)
}

// getTestRepository initializes the test database and returns a new UserRepository
// instance for testing. It also cleans the database before each test.
func getTestRepository(t *testing.T) UserRepository {
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewUserRepository(testDB)
}

// createTestUser creates a basic user in the database for testing purposes.
// Returns the created user's ID.
func createTestUser(t *testing.T, repo UserRepository) uint {
	user := &model.User{
		FirstName: "Test",
		LastName:  "User",
	}
	err := repo.Create(user)
	require.NoError(t, err)
	return user.ID
}

// UserRepository_Create tests the Create method of the UserRepository.
// It verifies that users are correctly persisted to the database with
// auto-generated IDs and timestamps.
func TestUserRepository_Create(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("create user sets ID and timestamps", func(t *testing.T) {
		user := &model.User{
			FirstName: "John",
			LastName:  "Doe",
		}

		err := repo.Create(user)

		assert.NoError(t, err)
		assert.NotZero(t, user.ID, "User ID should be auto-generated")
		assert.False(t, user.CreatedAt.IsZero(), "CreatedAt should be set")
		assert.False(t, user.UpdatedAt.IsZero(), "UpdatedAt should be set")

		// Verify user exists in database
		var found model.User
		dbErr := testDB.First(&found, user.ID).Error
		require.NoError(t, dbErr)
		assert.Equal(t, "John", found.FirstName)
		assert.Equal(t, "Doe", found.LastName)
	})

	t.Run("create user with all fields", func(t *testing.T) {
		profilePic := "http://example.com/pic.jpg"
		thumbnail := "http://example.com/thumb.jpg"
		timezone := "America/New_York"

		user := &model.User{
			FirstName:     "Jane",
			LastName:      "Smith",
			ProfilePicURL: &profilePic,
			ThumbnailURL:  &thumbnail,
			Timezone:      &timezone,
		}

		err := repo.Create(user)

		assert.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Verify all fields persisted
		var found model.User
		dbErr := testDB.First(&found, user.ID).Error
		require.NoError(t, dbErr)
		assert.Equal(t, "Jane", found.FirstName)
		assert.Equal(t, "Smith", found.LastName)
		assert.Equal(t, &profilePic, found.ProfilePicURL)
		assert.Equal(t, &thumbnail, found.ThumbnailURL)
		assert.Equal(t, &timezone, found.Timezone)
	})


}

// UserRepository_FindByID tests the FindByID method of the UserRepository.
// It verifies successful retrieval of existing users and proper error handling
// when users are not found.
func TestUserRepository_FindByID(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("find existing user by ID", func(t *testing.T) {
		userID := createTestUser(t, repo)

		found, err := repo.FindByID(userID)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, userID, found.ID)
		assert.Equal(t, "Test", found.FirstName)
		assert.Equal(t, "User", found.LastName)
	})

	t.Run("find non-existent user returns ErrUserNotFound", func(t *testing.T) {
		found, err := repo.FindByID(9999)

		assert.Error(t, err)
		assert.Nil(t, found)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("find deleted user returns ErrUserNotFound", func(t *testing.T) {
		userID := createTestUser(t, repo)

		// Soft delete the user
		err := repo.Delete(userID)
		require.NoError(t, err)

		// Attempt to find the deleted user
		found, err := repo.FindByID(userID)

		assert.Error(t, err)
		assert.Nil(t, found)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

// UserRepository_Update tests the Update method of the UserRepository.
// It verifies that partial updates are correctly applied to user records
// while leaving other fields unchanged.
func TestUserRepository_Update(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("update first name only", func(t *testing.T) {
		userID := createTestUser(t, repo)

		// Fetch the user to update
		user, err := repo.FindByID(userID)
		require.NoError(t, err)

		updates := map[string]interface{}{
			"first_name": "Updated",
		}

		err = repo.Update(user, updates)

		assert.NoError(t, err)

		// Verify update in database
		var found model.User
		dbErr := testDB.First(&found, userID).Error
		require.NoError(t, dbErr)
		assert.Equal(t, "Updated", found.FirstName)
		assert.Equal(t, "User", found.LastName) // Unchanged
	})

	t.Run("update multiple fields", func(t *testing.T) {
		userID := createTestUser(t, repo)
		profilePic := "http://example.com/new.jpg"
		timezone := "Europe/London"

		user, err := repo.FindByID(userID)
		require.NoError(t, err)

		updates := map[string]interface{}{
			"first_name":      "Multi",
			"last_name":       "Update",
			"profile_pic_url": profilePic,
			"timezone":        timezone,
		}

		err = repo.Update(user, updates)

		assert.NoError(t, err)

		// Verify all updates in database
		var found model.User
		dbErr := testDB.First(&found, userID).Error
		require.NoError(t, dbErr)
		assert.Equal(t, "Multi", found.FirstName)
		assert.Equal(t, "Update", found.LastName)
		assert.Equal(t, &profilePic, found.ProfilePicURL)
		assert.Equal(t, &timezone, found.Timezone)
	})

	t.Run("update timestamps are refreshed", func(t *testing.T) {
		userID := createTestUser(t, repo)

		user, err := repo.FindByID(userID)
		require.NoError(t, err)

		originalUpdatedAt := user.UpdatedAt
		// Small delay to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)

		updates := map[string]interface{}{
			"first_name": "Timestamp",
		}

		err = repo.Update(user, updates)
		require.NoError(t, err)

		// Fetch fresh from DB
		var found model.User
		dbErr := testDB.First(&found, userID).Error
		require.NoError(t, dbErr)
		assert.True(t, found.UpdatedAt.After(originalUpdatedAt), "UpdatedAt should be refreshed")
	})
}

// UserRepository_Delete tests the Delete method of the UserRepository.
// It verifies that users are soft-deleted (DeletedAt is set) and that
// attempting to delete non-existent users returns an error.
func TestUserRepository_Delete(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("delete existing user performs soft delete", func(t *testing.T) {
		userID := createTestUser(t, repo)

		err := repo.Delete(userID)

		assert.NoError(t, err)

		// Verify soft delete - record should still exist with DeletedAt set
		var user model.User
		dbErr := testDB.Unscoped().First(&user, userID).Error
		require.NoError(t, dbErr)
		assert.NotNil(t, user.DeletedAt)
		assert.False(t, user.DeletedAt.Time.IsZero())
	})

	t.Run("delete non-existent user returns ErrUserNotFound", func(t *testing.T) {
		err := repo.Delete(9999)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("delete already deleted user returns ErrUserNotFound", func(t *testing.T) {
		userID := createTestUser(t, repo)

		// First delete
		err := repo.Delete(userID)
		require.NoError(t, err)

		// Second delete should fail
		err = repo.Delete(userID)

		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

// UserRepository_FindAll tests the FindAll method of the UserRepository.
// It verifies retrieval of all non-deleted users and proper handling
// of empty result sets.
func TestUserRepository_FindAll(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("find all returns all non-deleted users", func(t *testing.T) {
		// Create multiple users
		user1 := &model.User{FirstName: "Alice", LastName: "Smith"}
		user2 := &model.User{FirstName: "Bob", LastName: "Jones"}
		user3 := &model.User{FirstName: "Carol", LastName: "White"}

		require.NoError(t, repo.Create(user1))
		require.NoError(t, repo.Create(user2))
		require.NoError(t, repo.Create(user3))

		users, err := repo.FindAll()

		assert.NoError(t, err)
		assert.Len(t, users, 3)

		// Verify all users are present
		names := make(map[string]bool)
		for _, u := range users {
			names[u.FirstName] = true
		}
		assert.True(t, names["Alice"])
		assert.True(t, names["Bob"])
		assert.True(t, names["Carol"])
	})

	t.Run("find all excludes soft-deleted users", func(t *testing.T) {
		cleanDatabase(t) // Ensure clean state for this subtest

		user1 := &model.User{FirstName: "Keep", LastName: "Me"}
		user2 := &model.User{FirstName: "Delete", LastName: "Me"}

		require.NoError(t, repo.Create(user1))
		require.NoError(t, repo.Create(user2))

		// Delete one user
		err := repo.Delete(user2.ID)
		require.NoError(t, err)

		users, err := repo.FindAll()

		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, "Keep", users[0].FirstName)
	})

	t.Run("find all returns empty slice when no users", func(t *testing.T) {
		cleanDatabase(t) // Ensure clean state for this subtest

		users, err := repo.FindAll()

		assert.NoError(t, err)
		assert.Empty(t, users)
		assert.NotNil(t, users) // Should be empty slice, not nil
	})

	t.Run("find all returns empty slice when all users deleted", func(t *testing.T) {
		cleanDatabase(t) // Ensure clean state for this subtest

		user := &model.User{FirstName: "Gone", LastName: "Soon"}
		require.NoError(t, repo.Create(user))

		// Delete all users
		err := repo.Delete(user.ID)
		require.NoError(t, err)

		users, err := repo.FindAll()

		assert.NoError(t, err)
		assert.Empty(t, users)
	})
}

// UserRepository_IntegrationWorkflow tests complex workflows combining
// multiple repository operations to verify they work together correctly.
func TestUserRepository_IntegrationWorkflow(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("full CRUD workflow", func(t *testing.T) {
		// Create
		user := &model.User{
			FirstName: "Workflow",
			LastName:  "Test",
		}
		err := repo.Create(user)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Read
		found, err := repo.FindByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Workflow", found.FirstName)

		// Update
		updates := map[string]interface{}{
			"first_name": "Updated",
		}
		err = repo.Update(found, updates)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.FindByID(user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.FirstName)

		// FindAll should include user
		allUsers, err := repo.FindAll()
		require.NoError(t, err)
		assert.Len(t, allUsers, 1)

		// Delete
		err = repo.Delete(user.ID)
		require.NoError(t, err)

		// Verify soft delete - should not be findable
		_, err = repo.FindByID(user.ID)
		assert.ErrorIs(t, err, ErrUserNotFound)

		// FindAll should be empty
		allUsers, err = repo.FindAll()
		require.NoError(t, err)
		assert.Empty(t, allUsers)
	})
}
