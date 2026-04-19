package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// activity_repository_test.go
//
// Subject: ActivityRepository (PostgreSQL/GORM) and ActivityRecordRepository (MongoDB)
// Scope:   Repository layer integration tests — CRUD operations, filtering, and aggregations.
// Out of scope:
//   - service-layer business rules   → activity_service_test.go
//   - HTTP contract validation       → activity_controller_test.go
// Infrastructure: PostgreSQL and MongoDB running locally (via docker-compose or local install)
// Data strategy: TRUNCATE PostgreSQL tables and DROP MongoDB collections between tests
// Parallel-safe: no — tests share database connections and clean state between runs

var (
	testDB      *gorm.DB
	testMongoDB *mongo.Database
	mongoClient *mongo.Client
	testLoc     *time.Location
	cfg         *config.Config
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	testLoc = time.UTC

	code := m.Run()

	if mongoClient != nil {
		_ = mongoClient.Disconnect(context.Background())
	}

	os.Exit(code)
}

func setupTestDatabases(t *testing.T) {
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

		// AutoMigrate creates the activities table for testing
		require.NoError(t, db.AutoMigrate(&model.Activity{}))
		testDB = db
	}

	if testMongoDB == nil {
		ctx := context.Background()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
		require.NoError(t, err)
		require.NoError(t, client.Ping(ctx, nil), "MongoDB not available")

		testMongoDB = client.Database(cfg.MongoDatabase)
		mongoClient = client
	}
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	require.NoError(t, testDB.Exec("TRUNCATE TABLE activities RESTART IDENTITY CASCADE;").Error)

	collections, _ := testMongoDB.ListCollectionNames(context.Background(), bson.M{})
	for _, coll := range collections {
		_ = testMongoDB.Collection(coll).Drop(context.Background())
	}
}

type testContext struct {
	activityRepo       ActivityRepository
	activityRecordRepo ActivityRecordRepository
	db                 *gorm.DB
	mongoDB            *mongo.Database
}

func getTestContext(t *testing.T) *testContext {
	setupTestDatabases(t)
	cleanDatabase(t)
	return &testContext{
		activityRepo:       NewActivityRepository(testDB),
		activityRecordRepo: NewActivityRecordRepository(testMongoDB),
		db:                 testDB,
		mongoDB:            testMongoDB,
	}
}

// ============================================================================
// PostgreSQL ActivityRepository Tests
// ============================================================================

// ActivityRepository_Create verifies that Create persists a new activity with
// auto-generated ID and timestamps. Covers basic happy path for activity creation.
func TestActivityRepository_Create(t *testing.T) {
	tc := getTestContext(t)

	activity := &model.Activity{
		UserID:           1,
		Title:            "Test Activity",
		Description:      "Test Description",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}

	err := tc.activityRepo.Create(activity)
	require.NoError(t, err)
	assert.NotZero(t, activity.ID, "Activity ID should be auto-generated")
	assert.False(t, activity.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, activity.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

// ActivityRepository_FindByID tests retrieval by primary key with user scoping.
// Success case: activity exists and belongs to user.
// Not found cases: wrong user ID, non-existent ID.
func TestActivityRepository_FindByID(t *testing.T) {
	tc := getTestContext(t)
	userID := uint(10)

	// Create test activity
	activity := &model.Activity{
		UserID:           userID,
		Title:            "Find Me",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(activity))

	t.Run("success - existing activity", func(t *testing.T) {
		found, err := tc.activityRepo.FindByID(activity.ID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, activity.Title, found.Title)
		assert.Equal(t, userID, found.UserID)
	})

	t.Run("not found - wrong user", func(t *testing.T) {
		found, err := tc.activityRepo.FindByID(activity.ID, 999)
		assert.ErrorIs(t, err, ErrActivityNotFound)
		assert.Nil(t, found)
	})

	t.Run("not found - non-existent ID", func(t *testing.T) {
		found, err := tc.activityRepo.FindByID(99999, userID)
		assert.ErrorIs(t, err, ErrActivityNotFound)
		assert.Nil(t, found)
	})
}

// ActivityRepository_FindByUserID tests the includeInactive filter behavior.
// When includeInactive=false: only active activities returned.
// When includeInactive=true: both active and inactive activities returned.
func TestActivityRepository_FindByUserID(t *testing.T) {
	tc := getTestContext(t)
	userID := uint(20)

	// Create active activity
	active := &model.Activity{
		UserID:           userID,
		Title:            "Active Activity",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(active))

	// Create inactive activity
	inactive := &model.Activity{
		UserID:           userID,
		Title:            "Inactive Activity",
		Frequency:        model.FrequencyWeekly,
		DayTime:          model.DayTimeEvening,
		CompletionAmount: 1,
		IsActive:         false,
	}
	require.NoError(t, tc.activityRepo.Create(inactive))

	// Activity for different user (should not appear)
	otherUser := &model.Activity{
		UserID:           999,
		Title:            "Other User Activity",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(otherUser))

	t.Run("include inactive false - returns only active", func(t *testing.T) {
		activities, err := tc.activityRepo.FindByUserID(userID, false)
		assert.NoError(t, err)
		assert.Len(t, activities, 1)
		assert.Equal(t, "Active Activity", activities[0].Title)
		assert.True(t, activities[0].IsActive)
	})

	t.Run("include inactive true - returns all", func(t *testing.T) {
		activities, err := tc.activityRepo.FindByUserID(userID, true)
		assert.NoError(t, err)
		assert.Len(t, activities, 2)
	})

	t.Run("different user returns empty", func(t *testing.T) {
		activities, err := tc.activityRepo.FindByUserID(888, false)
		assert.NoError(t, err)
		assert.Empty(t, activities)
	})
}

// ActivityRepository_FindActiveByUserID is a convenience method that should
// behave identically to FindByUserID with includeInactive=false.
func TestActivityRepository_FindActiveByUserID(t *testing.T) {
	tc := getTestContext(t)
	userID := uint(30)

	// Create active activity
	active := &model.Activity{
		UserID:           userID,
		Title:            "Active Activity",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(active))

	// Create inactive activity (should be filtered out)
	inactive := &model.Activity{
		UserID:           userID,
		Title:            "Inactive Activity",
		Frequency:        model.FrequencyWeekly,
		DayTime:          model.DayTimeEvening,
		CompletionAmount: 1,
		IsActive:         false,
	}
	require.NoError(t, tc.activityRepo.Create(inactive))

	activities, err := tc.activityRepo.FindActiveByUserID(userID)
	assert.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "Active Activity", activities[0].Title)
}

// ActivityRepository_FindFiltered tests the filter functionality with Frequency
// and DayTime parameters. Only active activities should be considered.
func TestActivityRepository_FindFiltered(t *testing.T) {
	tc := getTestContext(t)
	userID := uint(40)

	// Create activities with different frequencies and day times
	dailyMorning := &model.Activity{
		UserID:           userID,
		Title:            "Daily Morning",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(dailyMorning))

	dailyEvening := &model.Activity{
		UserID:           userID,
		Title:            "Daily Evening",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeEvening,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(dailyEvening))

	weeklyMorning := &model.Activity{
		UserID:           userID,
		Title:            "Weekly Morning",
		Frequency:        model.FrequencyWeekly,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(weeklyMorning))

	// Inactive activity should not appear in filtered results
	inactiveDaily := &model.Activity{
		UserID:           userID,
		Title:            "Inactive Daily",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         false,
	}
	require.NoError(t, tc.activityRepo.Create(inactiveDaily))

	t.Run("filter by frequency only", func(t *testing.T) {
		filter := &dto.ActivityFilter{Frequency: "daily"}
		activities, err := tc.activityRepo.FindFiltered(userID, filter)
		assert.NoError(t, err)
		assert.Len(t, activities, 2)
		for _, a := range activities {
			assert.Equal(t, model.FrequencyDaily, a.Frequency)
			assert.True(t, a.IsActive)
		}
	})

	t.Run("filter by day time only", func(t *testing.T) {
		filter := &dto.ActivityFilter{DayTime: "morning"}
		activities, err := tc.activityRepo.FindFiltered(userID, filter)
		assert.NoError(t, err)
		assert.Len(t, activities, 2)
		for _, a := range activities {
			assert.Equal(t, model.DayTimeMorning, a.DayTime)
			assert.True(t, a.IsActive)
		}
	})

	t.Run("filter by both frequency and day time", func(t *testing.T) {
		filter := &dto.ActivityFilter{
			Frequency: "daily",
			DayTime:   "morning",
		}
		activities, err := tc.activityRepo.FindFiltered(userID, filter)
		assert.NoError(t, err)
		assert.Len(t, activities, 1)
		assert.Equal(t, "Daily Morning", activities[0].Title)
	})

	t.Run("no filter - returns all active", func(t *testing.T) {
		filter := &dto.ActivityFilter{}
		activities, err := tc.activityRepo.FindFiltered(userID, filter)
		assert.NoError(t, err)
		assert.Len(t, activities, 3)
	})

	t.Run("filter with no matches", func(t *testing.T) {
		filter := &dto.ActivityFilter{Frequency: "monthly"}
		activities, err := tc.activityRepo.FindFiltered(userID, filter)
		assert.NoError(t, err)
		assert.Empty(t, activities)
	})
}

// ActivityRepository_Update tests partial updates via a map of fields.
// Verifies that specified fields are updated while others remain unchanged.
func TestActivityRepository_Update(t *testing.T) {
	tc := getTestContext(t)
	userID := uint(50)

	activity := &model.Activity{
		UserID:           userID,
		Title:            "Original Title",
		Description:      "Original Description",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(activity))
	originalTitle := activity.Title

	t.Run("update single field", func(t *testing.T) {
		updates := map[string]interface{}{
			"title": "Updated Title",
		}
		err := tc.activityRepo.Update(activity, updates)
		assert.NoError(t, err)

		// Reload and verify
		updated, err := tc.activityRepo.FindByID(activity.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", updated.Title)
		assert.Equal(t, "Original Description", updated.Description) // unchanged
	})

	t.Run("update multiple fields", func(t *testing.T) {
		// Reset to known state
		activity.Title = originalTitle
		activity.Description = "Original Description"
		activity.IsActive = true

		updates := map[string]interface{}{
			"title":       "New Title",
			"description": "New Description",
			"is_active":   false,
		}
		err := tc.activityRepo.Update(activity, updates)
		assert.NoError(t, err)

		updated, err := tc.activityRepo.FindByID(activity.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, "New Title", updated.Title)
		assert.Equal(t, "New Description", updated.Description)
		assert.False(t, updated.IsActive)
	})
}

// ActivityRepository_Delete tests soft deletion with user scoping.
// Success: activity marked as deleted (soft delete via DeletedAt).
// Not found: wrong user, already deleted activity.
func TestActivityRepository_Delete(t *testing.T) {
	tc := getTestContext(t)
	userID := uint(60)

	activity := &model.Activity{
		UserID:           userID,
		Title:            "To Be Deleted",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(activity))

	t.Run("successful deletion", func(t *testing.T) {
		err := tc.activityRepo.Delete(activity.ID, userID)
		assert.NoError(t, err)

		// Should not be findable via normal query
		found, err := tc.activityRepo.FindByID(activity.ID, userID)
		assert.ErrorIs(t, err, ErrActivityNotFound)
		assert.Nil(t, found)

		// But should exist in DB with DeletedAt set
		var deletedActivity model.Activity
		err = tc.db.Unscoped().First(&deletedActivity, activity.ID).Error
		require.NoError(t, err)
		assert.NotNil(t, deletedActivity.DeletedAt)
		assert.True(t, deletedActivity.DeletedAt.Valid)
	})

	t.Run("delete non-existent activity", func(t *testing.T) {
		err := tc.activityRepo.Delete(99999, userID)
		assert.ErrorIs(t, err, ErrActivityNotFound)
	})

	t.Run("delete with wrong user", func(t *testing.T) {
		// Create new activity to delete
		activity2 := &model.Activity{
			UserID:           userID,
			Title:            "Wrong User Test",
			Frequency:        model.FrequencyDaily,
			DayTime:          model.DayTimeMorning,
			CompletionAmount: 1,
			IsActive:         true,
		}
		require.NoError(t, tc.activityRepo.Create(activity2))

		err := tc.activityRepo.Delete(activity2.ID, 999)
		assert.ErrorIs(t, err, ErrActivityNotFound)

		// Verify activity still exists
		found, err := tc.activityRepo.FindByID(activity2.ID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
	})
}

// ============================================================================
// MongoDB ActivityRecordRepository Tests
// ============================================================================

// ActivityRecordRepository_Create verifies that Create persists a record with
// auto-generated ObjectID and timestamps.
func TestActivityRecordRepository_Create(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()

	record := &model.ActivityRecord{
		ActivityID:     1,
		UserID:         1,
		CompletionDate: time.Now().UTC(),
		Notes:          "Test completion",
		CreatedAt:      time.Now().UTC(),
	}

	err := tc.activityRecordRepo.Create(ctx, record)
	require.NoError(t, err)
	assert.NotEqual(t, primitive.NilObjectID, record.ID, "ID should be generated")
}

// ActivityRecordRepository_FindByActivityID tests retrieval with optional limit.
// Results should be sorted by completionDate descending (newest first).
func TestActivityRecordRepository_FindByActivityID(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(100)
	activityID := uint(1)

	// Create records at different times
	now := time.Now().UTC()
	records := []model.ActivityRecord{
		{ActivityID: activityID, UserID: userID, CompletionDate: now.Add(-2 * time.Hour), CreatedAt: now},
		{ActivityID: activityID, UserID: userID, CompletionDate: now.Add(-1 * time.Hour), CreatedAt: now},
		{ActivityID: activityID, UserID: userID, CompletionDate: now, CreatedAt: now},
	}
	for i := range records {
		require.NoError(t, tc.activityRecordRepo.Create(ctx, &records[i]))
	}

	// Create record for different activity (should not appear)
	otherRecord := model.ActivityRecord{
		ActivityID:     2,
		UserID:         userID,
		CompletionDate: now,
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &otherRecord))

	// Create record for different user (should not appear)
	otherUserRecord := model.ActivityRecord{
		ActivityID:     activityID,
		UserID:         999,
		CompletionDate: now,
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &otherUserRecord))

	t.Run("find all without limit", func(t *testing.T) {
		found, err := tc.activityRecordRepo.FindByActivityID(ctx, activityID, userID, 0)
		assert.NoError(t, err)
		assert.Len(t, found, 3)
		// Verify descending order by completion date
		assert.True(t, found[0].CompletionDate.After(found[1].CompletionDate))
	})

	t.Run("find with limit", func(t *testing.T) {
		found, err := tc.activityRecordRepo.FindByActivityID(ctx, activityID, userID, 2)
		assert.NoError(t, err)
		assert.Len(t, found, 2)
		// Should be the most recent ones
		assert.True(t, found[0].CompletionDate.After(found[1].CompletionDate))
	})

	t.Run("limit larger than count returns all", func(t *testing.T) {
		found, err := tc.activityRecordRepo.FindByActivityID(ctx, activityID, userID, 10)
		assert.NoError(t, err)
		assert.Len(t, found, 3)
	})

	t.Run("no matching records", func(t *testing.T) {
		found, err := tc.activityRecordRepo.FindByActivityID(ctx, 999, userID, 0)
		assert.NoError(t, err)
		assert.Empty(t, found)
	})
}

// ActivityRecordRepository_FindAll is a convenience wrapper around FindByActivityID
// with limit=0.
func TestActivityRecordRepository_FindAll(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(110)
	activityID := uint(2)

	// Create multiple records
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		record := model.ActivityRecord{
			ActivityID:     activityID,
			UserID:         userID,
			CompletionDate: now.Add(time.Duration(-i) * time.Hour),
			CreatedAt:      now,
		}
		require.NoError(t, tc.activityRecordRepo.Create(ctx, &record))
	}

	found, err := tc.activityRecordRepo.FindAll(ctx, activityID, userID)
	assert.NoError(t, err)
	assert.Len(t, found, 5)
}

// ActivityRecordRepository_CountByActivityID tests counting records scoped to
// both activity and user.
func TestActivityRecordRepository_CountByActivityID(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(120)
	activityID := uint(3)

	// Create 3 records for this activity
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		record := model.ActivityRecord{
			ActivityID:     activityID,
			UserID:         userID,
			CompletionDate: now.Add(time.Duration(-i) * time.Hour),
			CreatedAt:      now,
		}
		require.NoError(t, tc.activityRecordRepo.Create(ctx, &record))
	}

	// Create record for different activity
	otherActivity := model.ActivityRecord{
		ActivityID:     999,
		UserID:         userID,
		CompletionDate: now,
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &otherActivity))

	// Create record for different user
	otherUser := model.ActivityRecord{
		ActivityID:     activityID,
		UserID:         888,
		CompletionDate: now,
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &otherUser))

	t.Run("count existing records", func(t *testing.T) {
		count, err := tc.activityRecordRepo.CountByActivityID(ctx, activityID, userID)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})

	t.Run("count with no records", func(t *testing.T) {
		count, err := tc.activityRecordRepo.CountByActivityID(ctx, 88888, userID)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

// ActivityRecordRepository_DeleteLatestForDate tests deletion of the most recent
// completion within a date range. Used for undoing today's completion.
func TestActivityRecordRepository_DeleteLatestForDate(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(130)
	activityID := uint(4)
	loc := time.UTC

	// Create records for today at different times
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).UTC()
	endOfDay := startOfDay.Add(24 * time.Hour)

	olderRecord := model.ActivityRecord{
		ActivityID:     activityID,
		UserID:         userID,
		CompletionDate: startOfDay.Add(2 * time.Hour),
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &olderRecord))

	newestRecord := model.ActivityRecord{
		ActivityID:     activityID,
		UserID:         userID,
		CompletionDate: startOfDay.Add(14 * time.Hour),
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &newestRecord))

	// Create record for yesterday (outside range)
	yesterday := startOfDay.Add(-2 * time.Hour)
	yesterdayRecord := model.ActivityRecord{
		ActivityID:     activityID,
		UserID:         userID,
		CompletionDate: yesterday,
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &yesterdayRecord))

	t.Run("delete latest for date range", func(t *testing.T) {
		err := tc.activityRecordRepo.DeleteLatestForDate(ctx, activityID, userID, startOfDay, endOfDay)
		assert.NoError(t, err)

		// Should have deleted the newest record for today
		remaining, err := tc.activityRecordRepo.FindAll(ctx, activityID, userID)
		require.NoError(t, err)
		assert.Len(t, remaining, 2)

		// Verify the older today record still exists
		var foundNewest bool
		for _, r := range remaining {
			if r.ID == newestRecord.ID {
				foundNewest = true
				break
			}
		}
		assert.False(t, foundNewest, "Newest record should have been deleted")
	})

	t.Run("delete with no matching records", func(t *testing.T) {
		// Try to delete again (only one record left for today, already deleted)
		err := tc.activityRecordRepo.DeleteLatestForDate(ctx, activityID, userID, startOfDay, endOfDay)
		assert.ErrorIs(t, err, ErrActivityRecordNotFound)
	})

	t.Run("delete for wrong activity", func(t *testing.T) {
		err := tc.activityRecordRepo.DeleteLatestForDate(ctx, 99999, userID, startOfDay, endOfDay)
		assert.ErrorIs(t, err, ErrActivityRecordNotFound)
	})
}

// ActivityRecordRepository_GetCompletionMetadata tests the aggregation pipeline
// that returns completion counts, latest completions, and monthly completion info.
func TestActivityRecordRepository_GetCompletionMetadata(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(140)
	loc := time.UTC
	now := time.Now().In(loc)

	// Create activities in PostgreSQL
	activity1 := &model.Activity{
		UserID:           userID,
		Title:            "Activity 1",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(activity1))

	activity2 := &model.Activity{
		UserID:           userID,
		Title:            "Activity 2",
		Frequency:        model.FrequencyMonthly,
		DayTime:          model.DayTimeEvening,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(activity2))

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// Create records for today
	for i := 0; i < 3; i++ {
		record := model.ActivityRecord{
			ActivityID:     activity1.ID,
			UserID:         userID,
			CompletionDate: startOfDay.Add(time.Duration(i) * time.Hour),
			CreatedAt:      now,
		}
		require.NoError(t, tc.activityRecordRepo.Create(ctx, &record))
	}

	// Create record for yesterday (not in today count)
	yesterdayRecord := model.ActivityRecord{
		ActivityID:     activity2.ID,
		UserID:         userID,
		CompletionDate: startOfDay.Add(-24 * time.Hour),
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &yesterdayRecord))

	// Create record for last month (not in monthly completions)
	lastMonth := startOfDay.AddDate(0, -1, 0)
	lastMonthRecord := model.ActivityRecord{
		ActivityID:     activity1.ID,
		UserID:         userID,
		CompletionDate: lastMonth,
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &lastMonthRecord))

	t.Run("get metadata for multiple activities", func(t *testing.T) {
		activityIDs := []uint{activity1.ID, activity2.ID}
		metadata, err := tc.activityRecordRepo.GetCompletionMetadata(ctx, userID, activityIDs, loc)
		require.NoError(t, err)
		assert.NotNil(t, metadata)

		// Activity 1 has 3 completions today
		assert.Equal(t, 3, metadata.TodayCompletions[activity1.ID])

		// Activity 2 has no completions today
		assert.Equal(t, 0, metadata.TodayCompletions[activity2.ID])

		// Both have one-time completions
		assert.NotZero(t, metadata.OneTimeCompletions[activity1.ID])
		assert.NotZero(t, metadata.OneTimeCompletions[activity2.ID])

		// Only activity2 has a monthly completion (activity1's last completion was last month)
		_, hasMonthly1 := metadata.MonthlyCompletions[activity1.ID]
		_, hasMonthly2 := metadata.MonthlyCompletions[activity2.ID]
		assert.False(t, hasMonthly1, "Activity 1 should not have monthly completion")
		assert.True(t, hasMonthly2, "Activity 2 should have monthly completion")
	})

	t.Run("get metadata for non-existent activities", func(t *testing.T) {
		metadata, err := tc.activityRecordRepo.GetCompletionMetadata(ctx, userID, []uint{99999}, loc)
		require.NoError(t, err)
		assert.Empty(t, metadata.TodayCompletions)
		assert.Empty(t, metadata.MonthlyCompletions)
		assert.Empty(t, metadata.OneTimeCompletions)
	})
}

// ActivityRecordRepository_GetCompletionsForDate tests the aggregation that returns
// completion counts grouped by activity ID for a specific date.
func TestActivityRecordRepository_GetCompletionsForDate(t *testing.T) {
	tc := getTestContext(t)
	ctx := context.Background()
	userID := uint(150)
	loc := time.UTC
	now := time.Now().In(loc)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// Create activities
	activity1 := &model.Activity{
		UserID:           userID,
		Title:            "Activity 1",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(activity1))

	activity2 := &model.Activity{
		UserID:           userID,
		Title:            "Activity 2",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeEvening,
		CompletionAmount: 2,
		IsActive:         true,
	}
	require.NoError(t, tc.activityRepo.Create(activity2))

	// Create 3 completions for activity1 today
	for i := 0; i < 3; i++ {
		record := model.ActivityRecord{
			ActivityID:     activity1.ID,
			UserID:         userID,
			CompletionDate: startOfDay.Add(time.Duration(i) * time.Hour),
			CreatedAt:      now,
		}
		require.NoError(t, tc.activityRecordRepo.Create(ctx, &record))
	}

	// Create 1 completion for activity2 today
	record := model.ActivityRecord{
		ActivityID:     activity2.ID,
		UserID:         userID,
		CompletionDate: startOfDay.Add(5 * time.Hour),
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &record))

	// Create completion for yesterday (should not count)
	yesterdayRecord := model.ActivityRecord{
		ActivityID:     activity1.ID,
		UserID:         userID,
		CompletionDate: startOfDay.Add(-24 * time.Hour),
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &yesterdayRecord))

	// Create completion for different user (should not count)
	otherUserRecord := model.ActivityRecord{
		ActivityID:     activity1.ID,
		UserID:         999,
		CompletionDate: startOfDay.Add(2 * time.Hour),
		CreatedAt:      now,
	}
	require.NoError(t, tc.activityRecordRepo.Create(ctx, &otherUserRecord))

	t.Run("get completions for today", func(t *testing.T) {
		completions, err := tc.activityRecordRepo.GetCompletionsForDate(ctx, userID, now, loc)
		require.NoError(t, err)
		assert.Len(t, completions, 2)
		assert.Equal(t, 3, completions[activity1.ID])
		assert.Equal(t, 1, completions[activity2.ID])
	})

	t.Run("get completions for yesterday", func(t *testing.T) {
		yesterday := now.AddDate(0, 0, -1)
		completions, err := tc.activityRecordRepo.GetCompletionsForDate(ctx, userID, yesterday, loc)
		require.NoError(t, err)
		assert.Len(t, completions, 1)
		assert.Equal(t, 1, completions[activity1.ID])
	})

	t.Run("get completions for date with no records", func(t *testing.T) {
		lastWeek := now.AddDate(0, 0, -7)
		completions, err := tc.activityRecordRepo.GetCompletionsForDate(ctx, userID, lastWeek, loc)
		require.NoError(t, err)
		assert.Empty(t, completions)
	})
}
