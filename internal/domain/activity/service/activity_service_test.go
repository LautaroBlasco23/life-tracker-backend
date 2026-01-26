package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/model"
	"life-tracker-backend/internal/domain/activity/repository"

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
	service    *ActivityService
	db         *gorm.DB
	mongoDB    *mongo.Database
	recordRepo repository.ActivityRecordRepository
}

func getTestService(t *testing.T) *testContext {
	setupTestDatabases(t)
	cleanDatabase(t)
	return &testContext{
		service:    NewActivityService(testDB, testMongoDB),
		db:         testDB,
		mongoDB:    testMongoDB,
		recordRepo: repository.NewActivityRecordRepository(testMongoDB),
	}
}

func TestCreateActivity(t *testing.T) {
	tc := getTestService(t)

	tests := []struct {
		req     *dto.CreateActivityRequest
		name    string
		userID  uint
		wantErr bool
	}{
		{
			name:   "valid daily activity",
			userID: 10,
			req: &dto.CreateActivityRequest{
				Title:            "Morning Exercise",
				Description:      "30 min workout",
				CompletionAmount: 1,
				Frequency:        "daily",
				DayTime:          "morning",
			},
			wantErr: false,
		},
		{
			name:   "valid weekly activity",
			userID: 11,
			req: &dto.CreateActivityRequest{
				Title:            "Gym Session",
				CompletionAmount: 1,
				Frequency:        "weekly",
				DayFrequency:     `["monday","wednesday","friday"]`,
				DayTime:          "evening",
			},
			wantErr: false,
		},
		{
			name:   "invalid weekly activity - bad day frequency",
			userID: 12,
			req: &dto.CreateActivityRequest{
				Title:        "Invalid Activity",
				Frequency:    "weekly",
				DayFrequency: `["invalidday"]`,
				DayTime:      "morning",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tc.service.CreateActivity(tt.userID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.req.Title, resp.Title)
				assert.Equal(t, tt.userID, resp.UserID)
				assert.True(t, resp.IsActive)
			}
		})
	}
}

func TestGetUserActivities(t *testing.T) {
	tc := getTestService(t)
	userID := uint(100)

	active1 := model.Activity{
		UserID:           userID,
		Title:            "Active 1",
		IsActive:         true,
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
	}
	require.NoError(t, tc.db.Create(&active1).Error)

	inactive1 := model.Activity{
		UserID:           userID,
		Title:            "Inactive 1",
		IsActive:         false,
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
	}
	require.NoError(t, tc.db.Create(&inactive1).Error)

	otherUser := model.Activity{
		UserID:           2,
		Title:            "Other User",
		IsActive:         true,
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
	}
	require.NoError(t, tc.db.Create(&otherUser).Error)

	t.Run("get only active activities", func(t *testing.T) {
		resp, err := tc.service.GetUserActivities(userID, false, testLoc)
		assert.NoError(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, "Active 1", resp[0].Title)
	})

	t.Run("get all activities including inactive", func(t *testing.T) {
		resp, err := tc.service.GetUserActivities(userID, true, testLoc)
		assert.NoError(t, err)
		assert.Len(t, resp, 2)
	})

	t.Run("different user returns empty", func(t *testing.T) {
		resp, err := tc.service.GetUserActivities(999, false, testLoc)
		assert.NoError(t, err)
		assert.Empty(t, resp)
	})
}

func TestGetActivity(t *testing.T) {
	tc := getTestService(t)
	userID := uint(101)

	activity := model.Activity{
		UserID:           userID,
		Title:            "Test Activity",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		IsActive:         true,
		CompletionAmount: 1,
	}
	require.NoError(t, tc.db.Create(&activity).Error)

	t.Run("existing activity", func(t *testing.T) {
		resp, err := tc.service.GetActivity(userID, activity.ID)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, activity.Title, resp.Title)
	})

	t.Run("non-existent activity", func(t *testing.T) {
		resp, err := tc.service.GetActivity(userID, 9999)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("wrong user", func(t *testing.T) {
		resp, err := tc.service.GetActivity(999, activity.ID)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestUpdateActivity(t *testing.T) {
	tc := getTestService(t)
	userID := uint(102)

	activity := model.Activity{
		UserID:           userID,
		Title:            "Original Title",
		Description:      "Original Description",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.db.Create(&activity).Error)

	t.Run("update title and description", func(t *testing.T) {
		newTitle := "Updated Title"
		newDesc := "Updated Description"
		req := &dto.UpdateActivityRequest{
			Title:       &newTitle,
			Description: &newDesc,
		}

		resp, err := tc.service.UpdateActivity(userID, activity.ID, req)
		assert.NoError(t, err)
		assert.Equal(t, newTitle, resp.Title)
		assert.Equal(t, newDesc, resp.Description)
	})

	t.Run("deactivate activity", func(t *testing.T) {
		isActive := false
		req := &dto.UpdateActivityRequest{
			IsActive: &isActive,
		}

		resp, err := tc.service.UpdateActivity(userID, activity.ID, req)
		assert.NoError(t, err)
		assert.False(t, resp.IsActive)
	})

	t.Run("empty update returns unchanged", func(t *testing.T) {
		req := &dto.UpdateActivityRequest{}
		resp, err := tc.service.UpdateActivity(userID, activity.ID, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestDeleteActivity(t *testing.T) {
	tc := getTestService(t)
	userID := uint(103)

	activity := model.Activity{
		UserID:           userID,
		Title:            "To Delete",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.db.Create(&activity).Error)

	t.Run("successful deletion", func(t *testing.T) {
		err := tc.service.DeleteActivity(userID, activity.ID)
		assert.NoError(t, err)

		var deleted model.Activity
		err = tc.db.Unscoped().First(&deleted, activity.ID).Error
		assert.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
	})

	t.Run("delete non-existent activity", func(t *testing.T) {
		err := tc.service.DeleteActivity(userID, 9999)
		assert.Error(t, err)
	})
}

func TestRecordActivity(t *testing.T) {
	tc := getTestService(t)
	userID := uint(104)

	activity := model.Activity{
		UserID:           userID,
		Title:            "Record Test",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.db.Create(&activity).Error)

	t.Run("record with current time", func(t *testing.T) {
		req := &dto.RecordActivityRequest{
			Notes: "Completed successfully",
		}

		resp, err := tc.service.RecordActivity(userID, activity.ID, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, activity.ID, resp.ActivityID)
		assert.Equal(t, "Completed successfully", resp.Notes)
	})

	t.Run("record with custom date", func(t *testing.T) {
		customDate := time.Now().Add(-24 * time.Hour)
		req := &dto.RecordActivityRequest{
			CompletionDate: customDate,
			Notes:          "Yesterday",
		}

		resp, err := tc.service.RecordActivity(userID, activity.ID, req)
		assert.NoError(t, err)
		assert.WithinDuration(t, customDate, resp.CompletionDate, time.Second)
	})

	t.Run("record for non-existent activity", func(t *testing.T) {
		req := &dto.RecordActivityRequest{}
		resp, err := tc.service.RecordActivity(userID, 9999, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestValidateDayFrequency(t *testing.T) {
	tc := getTestService(t)

	tests := []struct {
		name    string
		dayFreq string
		wantErr bool
	}{
		{
			name:    "valid days",
			dayFreq: `["monday","wednesday","friday"]`,
			wantErr: false,
		},
		{
			name:    "all days",
			dayFreq: `["monday","tuesday","wednesday","thursday","friday","saturday","sunday"]`,
			wantErr: false,
		},
		{
			name:    "invalid day",
			dayFreq: `["monday","notaday"]`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			dayFreq: `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tc.service.validateDayFrequency(tt.dayFreq)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestShouldShowToday(t *testing.T) {
	tc := getTestService(t)

	now := time.Now().In(testLoc)
	metadata := &repository.CompletionMetadata{
		MonthlyCompletions: make(map[uint]time.Time),
		OneTimeCompletions: make(map[uint]time.Time),
		TodayCompletions:   make(map[uint]int),
	}

	tests := []struct {
		setup    func()
		name     string
		activity model.Activity
		want     bool
	}{
		{
			name: "daily activity always shown",
			activity: model.Activity{
				ID:        1,
				Frequency: model.FrequencyDaily,
			},
			setup: func() {},
			want:  true,
		},
		{
			name: "weekly activity on correct day",
			activity: model.Activity{
				ID:        2,
				Frequency: model.FrequencyWeekly,
				DayFrequency: func() string {
					days := []string{strings.ToLower(now.Weekday().String())}
					b, _ := json.Marshal(days)
					return string(b)
				}(),
			},
			setup: func() {},
			want:  true,
		},
		{
			name: "weekly activity on wrong day",
			activity: model.Activity{
				ID:           3,
				Frequency:    model.FrequencyWeekly,
				DayFrequency: `["monday"]`,
			},
			setup: func() {},
			want:  now.Weekday() == time.Monday,
		},
		{
			name: "monthly not completed this month",
			activity: model.Activity{
				ID:        4,
				Frequency: model.FrequencyMonthly,
			},
			setup: func() {},
			want:  true,
		},
		{
			name: "monthly already completed",
			activity: model.Activity{
				ID:        5,
				Frequency: model.FrequencyMonthly,
			},
			setup: func() {
				metadata.MonthlyCompletions[5] = time.Now()
			},
			want: false,
		},
		{
			name: "one-time never completed",
			activity: model.Activity{
				ID:        6,
				Frequency: model.FrequencyOneTime,
			},
			setup: func() {},
			want:  true,
		},
		{
			name: "one-time completed today",
			activity: model.Activity{
				ID:        7,
				Frequency: model.FrequencyOneTime,
			},
			setup: func() {
				metadata.OneTimeCompletions[7] = time.Now()
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata.MonthlyCompletions = make(map[uint]time.Time)
			metadata.OneTimeCompletions = make(map[uint]time.Time)
			tt.setup()

			got := tc.service.shouldShowToday(&tt.activity, metadata, now)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRevertLastCompletion(t *testing.T) {
	tc := getTestService(t)
	userID := uint(105)

	activity := model.Activity{
		UserID:           userID,
		Title:            "Revert Test",
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
		IsActive:         true,
	}
	require.NoError(t, tc.db.Create(&activity).Error)

	t.Run("revert existing completion", func(t *testing.T) {
		record := model.ActivityRecord{
			ID:             primitive.NewObjectID(),
			ActivityID:     activity.ID,
			UserID:         userID,
			CompletionDate: time.Now(),
			CreatedAt:      time.Now(),
		}

		collection := tc.mongoDB.Collection("activity_records")
		_, err := collection.InsertOne(context.Background(), record)
		require.NoError(t, err)

		err = tc.service.RevertLastCompletion(userID, activity.ID, nil, testLoc)
		assert.NoError(t, err)

		count, err := collection.CountDocuments(context.Background(), bson.M{"_id": record.ID})
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("revert with no completions", func(t *testing.T) {
		err := tc.service.RevertLastCompletion(userID, activity.ID, nil, testLoc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no completion found")
	})
}

func TestGetTodayActivities(t *testing.T) {
	tc := getTestService(t)
	userID := uint(106)

	daily := model.Activity{
		UserID:           userID,
		Title:            "Daily Activity",
		IsActive:         true,
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
	}
	require.NoError(t, tc.db.Create(&daily).Error)

	t.Run("returns daily activities", func(t *testing.T) {
		resp, err := tc.service.GetTodayActivities(userID, testLoc)
		assert.NoError(t, err)
		assert.Len(t, resp, 1)
		assert.Equal(t, "Daily Activity", resp[0].Title)
	})
}

func TestGetUserActivitiesFiltered(t *testing.T) {
	tc := getTestService(t)
	userID := uint(107)

	activity := model.Activity{
		UserID:           userID,
		Title:            "Filtered Activity",
		IsActive:         true,
		Frequency:        model.FrequencyDaily,
		DayTime:          model.DayTimeMorning,
		CompletionAmount: 1,
	}
	require.NoError(t, tc.db.Create(&activity).Error)

	t.Run("filter by frequency", func(t *testing.T) {
		filter := &dto.ActivityFilter{
			Frequency: "daily",
		}
		resp, err := tc.service.GetUserActivitiesFiltered(userID, filter, testLoc)
		assert.NoError(t, err)
		assert.Len(t, resp, 1)
	})

	t.Run("filter by day time", func(t *testing.T) {
		filter := &dto.ActivityFilter{
			DayTime: "morning",
		}
		resp, err := tc.service.GetUserActivitiesFiltered(userID, filter, testLoc)
		assert.NoError(t, err)
		assert.Len(t, resp, 1)
	})
}
