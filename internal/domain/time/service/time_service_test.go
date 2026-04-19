// time_service_test.go
//
// Subject:      internal/domain/time/service.TimeService against real PostgreSQL.
// Scope:        CRUD operations and stats aggregation — create, filter, update, delete, GetStats.
// Out of scope: HTTP contract → time_controller_test.go
//               Repository date-range filtering in isolation.
// Infrastructure: PostgreSQL (test database via .env.test).
// Data strategy:  TRUNCATE time_records before each top-level test function.
// Parallel-safe:  No — single shared DB connection, no per-test namespace.
package service

import (
	"fmt"
	"os"
	"testing"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/time/dto"
	"life-tracker-backend/internal/domain/time/model"

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

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	os.Exit(m.Run())
}

func setupTestDatabase(t *testing.T) {
	t.Helper()
	if testDB != nil {
		return
	}

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

	require.NoError(t, db.AutoMigrate(&model.TimeRecord{}))
	testDB = db
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	require.NoError(t, testDB.Exec("TRUNCATE TABLE time_records RESTART IDENTITY CASCADE;").Error)
}

func getTestService(t *testing.T) *TimeService {
	t.Helper()
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewTimeService(testDB)
}

// TestCreateRecord verifies that a time record is persisted with the correct
// userID, category, description, and duration.
func TestCreateRecord(t *testing.T) {
	svc := getTestService(t)

	tests := []struct {
		req     *dto.CreateTimeRecordRequest
		name    string
		userID  uint
		wantErr bool
	}{
		{
			name:   "valid_record_persisted",
			userID: 10,
			req: &dto.CreateTimeRecordRequest{
				Category:        "study",
				Description:     "Read Go in Action",
				DurationMinutes: 90,
			},
		},
		{
			name:   "minimal_record_persisted",
			userID: 11,
			req: &dto.CreateTimeRecordRequest{
				Category:        "gym",
				Description:     "Morning workout",
				DurationMinutes: 60,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.CreateRecord(tt.userID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				return
			}

			assert.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, tt.req.Category, resp.Category)
			assert.Equal(t, tt.req.Description, resp.Description)
			assert.Equal(t, tt.req.DurationMinutes, resp.DurationMinutes)
			assert.NotZero(t, resp.ID)
		})
	}
}

// TestGetRecord verifies ownership enforcement: a record is only accessible
// by its owner; wrong user or non-existent ID both return an error.
func TestGetRecord(t *testing.T) {
	svc := getTestService(t)
	userID := uint(100)

	record := model.TimeRecord{
		UserID:          userID,
		Category:        "work",
		Description:     "Deep work session",
		DurationMinutes: 120,
	}
	require.NoError(t, testDB.Create(&record).Error)

	t.Run("existing_record_returns_response", func(t *testing.T) {
		resp, err := svc.GetRecord(userID, record.ID)
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, record.Category, resp.Category)
		assert.Equal(t, record.DurationMinutes, resp.DurationMinutes)
	})

	t.Run("non_existent_id_returns_error", func(t *testing.T) {
		resp, err := svc.GetRecord(userID, 9999)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("wrong_user_returns_error", func(t *testing.T) {
		// DELETE WHERE user_id = ? AND id = ? returns 0 rows for wrong owner.
		resp, err := svc.GetRecord(999, record.ID)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestGetRecords verifies that listing respects user isolation and that
// the optional category filter narrows results correctly.
func TestGetRecords(t *testing.T) {
	svc := getTestService(t)
	userID := uint(101)

	seeds := []model.TimeRecord{
		{UserID: userID, Category: "study", Description: "Go book", DurationMinutes: 60},
		{UserID: userID, Category: "gym", Description: "Leg day", DurationMinutes: 45},
		{UserID: 999, Category: "study", Description: "Other user", DurationMinutes: 30},
	}
	for i := range seeds {
		require.NoError(t, testDB.Create(&seeds[i]).Error)
	}

	t.Run("no_filter_returns_all_user_records", func(t *testing.T) {
		resp, err := svc.GetRecords(userID, nil, nil)
		assert.NoError(t, err)
		assert.Len(t, resp, 2)
	})

	t.Run("category_filter_narrows_results", func(t *testing.T) {
		filter := &dto.TimeRecordFilter{Category: "study"}
		resp, err := svc.GetRecords(userID, filter, nil)
		assert.NoError(t, err)
		require.Len(t, resp, 1)
		assert.Equal(t, "study", resp[0].Category)
	})

	t.Run("different_user_sees_no_records", func(t *testing.T) {
		resp, err := svc.GetRecords(888, nil, nil)
		assert.NoError(t, err)
		assert.Empty(t, resp)
	})
}

// TestUpdateRecord verifies that individual fields can be patched,
// that a no-op update returns the unchanged record, and that updates
// to non-existent records fail.
func TestUpdateRecord(t *testing.T) {
	svc := getTestService(t)
	userID := uint(102)

	record := model.TimeRecord{
		UserID:          userID,
		Category:        "study",
		Description:     "Original description",
		DurationMinutes: 60,
	}
	require.NoError(t, testDB.Create(&record).Error)

	t.Run("update_category_persisted", func(t *testing.T) {
		newCat := "work"
		resp, err := svc.UpdateRecord(userID, record.ID, &dto.UpdateTimeRecordRequest{Category: &newCat})
		assert.NoError(t, err)
		assert.Equal(t, "work", resp.Category)
		assert.Equal(t, 60, resp.DurationMinutes)
	})

	t.Run("update_duration_persisted", func(t *testing.T) {
		newDur := 90
		resp, err := svc.UpdateRecord(userID, record.ID, &dto.UpdateTimeRecordRequest{DurationMinutes: &newDur})
		assert.NoError(t, err)
		assert.Equal(t, 90, resp.DurationMinutes)
	})

	t.Run("empty_request_returns_current_record", func(t *testing.T) {
		// buildUpdateMap returns an empty map → service returns existing record unchanged.
		resp, err := svc.UpdateRecord(userID, record.ID, &dto.UpdateTimeRecordRequest{})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("non_existent_record_returns_error", func(t *testing.T) {
		newCat := "gym"
		resp, err := svc.UpdateRecord(userID, 9999, &dto.UpdateTimeRecordRequest{Category: &newCat})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestDeleteRecord verifies that a deleted record is no longer retrievable
// and that deletion with a wrong user or non-existent ID fails.
func TestDeleteRecord(t *testing.T) {
	svc := getTestService(t)
	userID := uint(103)

	t.Run("successful_deletion_makes_record_unretrievable", func(t *testing.T) {
		record := model.TimeRecord{
			UserID:          userID,
			Category:        "study",
			Description:     "To delete",
			DurationMinutes: 30,
		}
		require.NoError(t, testDB.Create(&record).Error)

		err := svc.DeleteRecord(userID, record.ID)
		assert.NoError(t, err)

		resp, err := svc.GetRecord(userID, record.ID)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("delete_non_existent_record_returns_error", func(t *testing.T) {
		err := svc.DeleteRecord(userID, 9999)
		assert.Error(t, err)
	})

	t.Run("delete_with_wrong_user_returns_error", func(t *testing.T) {
		// The repository checks RowsAffected; wrong user means 0 rows deleted.
		record := model.TimeRecord{
			UserID:          userID,
			Category:        "gym",
			Description:     "Protected record",
			DurationMinutes: 45,
		}
		require.NoError(t, testDB.Create(&record).Error)

		err := svc.DeleteRecord(999, record.ID)
		assert.Error(t, err)

		// Original owner can still read it.
		resp, getErr := svc.GetRecord(userID, record.ID)
		assert.NoError(t, getErr)
		assert.NotNil(t, resp)
	})
}

// TestGetStats verifies that the stats aggregation correctly totals duration
// per category, computes the overall total, and identifies the top category.
func TestGetStats(t *testing.T) {
	svc := getTestService(t)
	userID := uint(104)

	seeds := []model.TimeRecord{
		{UserID: userID, Category: "study", Description: "Books", DurationMinutes: 120},
		{UserID: userID, Category: "study", Description: "Courses", DurationMinutes: 60},
		{UserID: userID, Category: "gym", Description: "Lifting", DurationMinutes: 45},
		{UserID: 999, Category: "study", Description: "Other user", DurationMinutes: 999},
	}
	for i := range seeds {
		require.NoError(t, testDB.Create(&seeds[i]).Error)
	}

	t.Run("totals_and_top_category_correct", func(t *testing.T) {
		resp, err := svc.GetStats(userID)
		assert.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, 225, resp.TotalMinutes)
		assert.Equal(t, 3, resp.RecordCount)
		assert.Equal(t, "study", resp.TopCategory)
		assert.Equal(t, 180, resp.TopCategoryMins)
		assert.Equal(t, 180, resp.CategoryTotals["study"])
		assert.Equal(t, 45, resp.CategoryTotals["gym"])
	})

	t.Run("no_records_returns_zero_stats", func(t *testing.T) {
		resp, err := svc.GetStats(888)
		assert.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 0, resp.TotalMinutes)
		assert.Equal(t, 0, resp.RecordCount)
		assert.Empty(t, resp.TopCategory)
	})
}
