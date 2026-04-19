package repository

import (
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/time/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// time_repository_integration_test.go
//
// Subject: internal/domain/time/repository.TimeRecordRepository against real PostgreSQL.
// Scope:   SQL correctness for time record CRUD operations, filtering, and ownership validation.
// Out of scope:
//   - service-layer business rules  → time_service_test.go (unit)
//   - HTTP contract                 → time_controller_test.go (unit)
// Infrastructure: PostgreSQL (test database), migrations via AutoMigrate.
// Data strategy: TRUNCATE TABLE after each test, single test DB connection.
// Parallel-safe: no — shared testDB with sequential cleanup.

var (
	testDB *gorm.DB
	cfg    *config.Config
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	code := m.Run()
	os.Exit(code)
}

func setupTestDatabase(t *testing.T) {
	t.Helper()
	if testDB == nil {
		dsn := "host=" + cfg.DBHost + " port=" + cfg.DBPort + " user=" + cfg.DBUser +
			" password=" + cfg.DBPassword + " dbname=" + cfg.DBName + " sslmode=" + cfg.DBSSLMode

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		require.NoError(t, err, "Failed to connect to PostgreSQL")

		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.SetMaxOpenConns(1)

		require.NoError(t, db.AutoMigrate(&model.TimeRecord{}))
		testDB = db
	}
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	require.NoError(t, testDB.Exec("TRUNCATE TABLE time_records RESTART IDENTITY CASCADE;").Error)
}

func getTestRepository(t *testing.T) TimeRecordRepository {
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewTimeRecordRepository(testDB)
}

func createTestTimeRecord(t *testing.T, repo TimeRecordRepository, userID uint, category string, duration int, description string) *model.TimeRecord {
	t.Helper()
	record := &model.TimeRecord{
		UserID:          userID,
		Category:        category,
		DurationMinutes: duration,
		Description:     description,
	}
	err := repo.Create(record)
	require.NoError(t, err)
	require.NotZero(t, record.ID)
	return record
}

// TimeRecordRepository_Create tests the Create method.
// Covers: basic creation with all fields populated, ID auto-generation.
func TestTimeRecordRepository_Create(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("create time record with all fields", func(t *testing.T) {
		record := &model.TimeRecord{
			UserID:          1,
			Category:        "Work",
			DurationMinutes: 60,
			Description:     "Completed project tasks",
		}

		err := repo.Create(record)

		assert.NoError(t, err)
		assert.NotZero(t, record.ID)
		assert.NotZero(t, record.CreatedAt)
		assert.NotZero(t, record.UpdatedAt)

		// Verify record exists in database
		var found model.TimeRecord
		err = testDB.First(&found, record.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "Work", found.Category)
		assert.Equal(t, 60, found.DurationMinutes)
		assert.Equal(t, "Completed project tasks", found.Description)
	})

	t.Run("create minimal time record", func(t *testing.T) {
		record := &model.TimeRecord{
			UserID:          2,
			Category:        "Exercise",
			DurationMinutes: 30,
		}

		err := repo.Create(record)

		assert.NoError(t, err)
		assert.NotZero(t, record.ID)
	})
}

// TimeRecordRepository_FindByID tests record retrieval by ID with user ownership check.
// Covers: existing record retrieval, not-found error for non-existent ID, not-found error for wrong user.
func TestTimeRecordRepository_FindByID(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("find existing record with correct user", func(t *testing.T) {
		created := createTestTimeRecord(t, repo, 1, "Work", 60, "Test description")

		found, err := repo.FindByID(created.ID, 1)

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, "Work", found.Category)
		assert.Equal(t, 60, found.DurationMinutes)
		assert.Equal(t, "Test description", found.Description)
		assert.Equal(t, uint(1), found.UserID)
	})

	t.Run("returns ErrTimeRecordNotFound for non-existent record", func(t *testing.T) {
		found, err := repo.FindByID(9999, 1)

		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Equal(t, ErrTimeRecordNotFound, err)
	})

	t.Run("returns ErrTimeRecordNotFound for wrong user", func(t *testing.T) {
		created := createTestTimeRecord(t, repo, 1, "Work", 60, "Test description")

		// Try to access with different user ID
		found, err := repo.FindByID(created.ID, 2)

		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Equal(t, ErrTimeRecordNotFound, err)
	})
}

// TimeRecordRepository_FindByUserID tests filtered retrieval of user records.
// Covers: no filter (all records), category filter, date range filter, combined filters.
func TestTimeRecordRepository_FindByUserID(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("with no filter returns all user records", func(t *testing.T) {
		// Create records for user 1
		createTestTimeRecord(t, repo, 1, "Work", 60, "Task 1")
		createTestTimeRecord(t, repo, 1, "Exercise", 30, "Task 2")
		createTestTimeRecord(t, repo, 1, "Reading", 45, "Task 3")
		// Create record for user 2 (should not appear)
		createTestTimeRecord(t, repo, 2, "Work", 90, "Other user task")

		records, err := repo.FindByUserID(1, nil)

		assert.NoError(t, err)
		assert.Len(t, records, 3)
		// Should be ordered by created_at DESC
		assert.Equal(t, "Reading", records[0].Category)
		assert.Equal(t, "Exercise", records[1].Category)
		assert.Equal(t, "Work", records[2].Category)
	})

	t.Run("with Category filter", func(t *testing.T) {
		createTestTimeRecord(t, repo, 1, "Work", 60, "Work task")
		createTestTimeRecord(t, repo, 1, "Exercise", 30, "Exercise task")
		createTestTimeRecord(t, repo, 1, "Work", 90, "Another work task")

		filter := &TimeRecordFilter{
			Category: "Work",
		}
		records, err := repo.FindByUserID(1, filter)

		assert.NoError(t, err)
		assert.Len(t, records, 2)
		for _, r := range records {
			assert.Equal(t, "Work", r.Category)
		}
	})

	t.Run("with date range filter", func(t *testing.T) {
		// Create records at specific times
		now := time.Now().UTC()

		record1 := &model.TimeRecord{
			UserID:          1,
			Category:        "Work",
			DurationMinutes: 60,
		}
		err := repo.Create(record1)
		require.NoError(t, err)

		// Manually update created_at to be in the past
		pastTime := now.Add(-7 * 24 * time.Hour)
		err = testDB.Model(record1).Update("created_at", pastTime).Error
		require.NoError(t, err)

		record2 := &model.TimeRecord{
			UserID:          1,
			Category:        "Exercise",
			DurationMinutes: 30,
		}
		err = repo.Create(record2)
		require.NoError(t, err)

		// Filter for last 3 days only (should only get record2)
		startDate := now.Add(-3 * 24 * time.Hour)
		endDate := now.Add(24 * time.Hour)
		filter := &TimeRecordFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		records, err := repo.FindByUserID(1, filter)

		assert.NoError(t, err)
		assert.Len(t, records, 1)
		assert.Equal(t, "Exercise", records[0].Category)
	})

	t.Run("with combined filters", func(t *testing.T) {
		now := time.Now().UTC()

		// Create Work record in the past
		record1 := &model.TimeRecord{
			UserID:          1,
			Category:        "Work",
			DurationMinutes: 60,
		}
		err := repo.Create(record1)
		require.NoError(t, err)
		pastTime := now.Add(-7 * 24 * time.Hour)
		err = testDB.Model(record1).Update("created_at", pastTime).Error
		require.NoError(t, err)

		// Create Work record recently
		record2 := &model.TimeRecord{
			UserID:          1,
			Category:        "Work",
			DurationMinutes: 90,
		}
		err = repo.Create(record2)
		require.NoError(t, err)

		// Create Exercise record recently
		record3 := &model.TimeRecord{
			UserID:          1,
			Category:        "Exercise",
			DurationMinutes: 30,
		}
		err = repo.Create(record3)
		require.NoError(t, err)

		// Filter for Work category in last 3 days
		startDate := now.Add(-3 * 24 * time.Hour)
		endDate := now.Add(24 * time.Hour)
		filter := &TimeRecordFilter{
			Category:  "Work",
			StartDate: &startDate,
			EndDate:   &endDate,
		}
		records, err := repo.FindByUserID(1, filter)

		assert.NoError(t, err)
		assert.Len(t, records, 1)
		assert.Equal(t, "Work", records[0].Category)
		assert.Equal(t, 90, records[0].DurationMinutes)
	})

	t.Run("returns empty slice when no records match", func(t *testing.T) {
		createTestTimeRecord(t, repo, 1, "Work", 60, "Task")

		filter := &TimeRecordFilter{
			Category: "NonExistent",
		}
		records, err := repo.FindByUserID(1, filter)

		assert.NoError(t, err)
		assert.Empty(t, records)
	})
}

// TimeRecordRepository_Update tests partial updates to time records.
// Covers: updating individual fields, multiple fields at once.
func TestTimeRecordRepository_Update(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("update time record fields", func(t *testing.T) {
		created := createTestTimeRecord(t, repo, 1, "Work", 60, "Original description")

		updates := map[string]interface{}{
			"category":        "Exercise",
			"duration_minutes": 90,
			"description":     "Updated description",
		}

		err := repo.Update(created, updates)

		assert.NoError(t, err)

		// Verify updates
		updated, err := repo.FindByID(created.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, "Exercise", updated.Category)
		assert.Equal(t, 90, updated.DurationMinutes)
		assert.Equal(t, "Updated description", updated.Description)
	})

	t.Run("update single field", func(t *testing.T) {
		created := createTestTimeRecord(t, repo, 1, "Work", 60, "Description")

		updates := map[string]interface{}{
			"duration_minutes": 120,
		}

		err := repo.Update(created, updates)

		assert.NoError(t, err)

		updated, err := repo.FindByID(created.ID, 1)
		require.NoError(t, err)
		assert.Equal(t, 120, updated.DurationMinutes)
		assert.Equal(t, "Work", updated.Category) // Unchanged
		assert.Equal(t, "Description", updated.Description) // Unchanged
	})
}

// TimeRecordRepository_Delete tests record deletion with ownership check.
// Covers: successful deletion, not-found error for non-existent record.
func TestTimeRecordRepository_Delete(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("delete existing record", func(t *testing.T) {
		created := createTestTimeRecord(t, repo, 1, "Work", 60, "To be deleted")

		err := repo.Delete(created.ID, 1)

		assert.NoError(t, err)

		// Verify record is gone
		_, err = repo.FindByID(created.ID, 1)
		assert.Equal(t, ErrTimeRecordNotFound, err)
	})

	t.Run("delete non-existent returns ErrTimeRecordNotFound", func(t *testing.T) {
		err := repo.Delete(9999, 1)

		assert.Error(t, err)
		assert.Equal(t, ErrTimeRecordNotFound, err)
	})

	t.Run("delete with wrong user returns ErrTimeRecordNotFound", func(t *testing.T) {
		created := createTestTimeRecord(t, repo, 1, "Work", 60, "Protected")

		err := repo.Delete(created.ID, 2)

		assert.Error(t, err)
		assert.Equal(t, ErrTimeRecordNotFound, err)

		// Verify record still exists
		found, err := repo.FindByID(created.ID, 1)
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}

// TimeRecordRepository_FindAllByUserID tests retrieval of all records for a user without filtering.
// Covers: returns all records for user, returns empty for user with no records.
func TestTimeRecordRepository_FindAllByUserID(t *testing.T) {
	repo := getTestRepository(t)

	t.Run("returns all records for user", func(t *testing.T) {
		createTestTimeRecord(t, repo, 1, "Work", 60, "Task 1")
		createTestTimeRecord(t, repo, 1, "Exercise", 30, "Task 2")
		createTestTimeRecord(t, repo, 1, "Reading", 45, "Task 3")
		// Create record for user 2
		createTestTimeRecord(t, repo, 2, "Work", 90, "Other user")

		records, err := repo.FindAllByUserID(1)

		assert.NoError(t, err)
		assert.Len(t, records, 3)
	})

	t.Run("returns empty slice for user with no records", func(t *testing.T) {
		createTestTimeRecord(t, repo, 1, "Work", 60, "Task")

		records, err := repo.FindAllByUserID(999)

		assert.NoError(t, err)
		assert.Empty(t, records)
	})
}
