package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/domain/activity/dto"
	"life-tracker-backend/internal/domain/activity/model"
	"life-tracker-backend/internal/domain/activity/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *service.ActivityService, func()) {
	gin.SetMode(gin.TestMode)

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	require.NotEmpty(t, dsn, "TEST_POSTGRES_DSN environment variable must be set. Run tests with 'make test'")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to PostgreSQL. Is the container running?")

	db.Exec("DROP TABLE IF EXISTS activities CASCADE")
	require.NoError(t, db.AutoMigrate(&model.Activity{}))

	uri := os.Getenv("TEST_MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	require.NoError(t, err)
	require.NoError(t, client.Ping(context.Background(), nil), "MongoDB not available")

	dbName := "test_controller_" + primitive.NewObjectID().Hex()
	mongoDB := client.Database(dbName)

	activityService := service.NewActivityService(db, mongoDB)
	controller := NewActivityController(activityService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	activities := router.Group("/activities")
	{
		activities.POST("", controller.CreateActivity)
		activities.GET("", controller.GetUserActivities)
		activities.GET("/today", controller.GetTodayActivities)
		activities.GET("/:id", controller.GetActivity)
		activities.PUT("/:id", controller.UpdateActivity)
		activities.DELETE("/:id", controller.DeleteActivity)
		activities.POST("/:id/record", controller.RecordActivity)
		activities.GET("/:id/records", controller.GetActivityRecords)
		activities.GET("/:id/stats", controller.GetActivityStats)
		activities.DELETE("/:id/revert", controller.RevertLastCompletion)
	}

	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS activities CASCADE")
		_ = mongoDB.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}

	return router, activityService, cleanup
}

func TestCreateActivityHandler(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	tests := []struct {
		body           interface{}
		checkResponse  func(*testing.T, map[string]interface{})
		name           string
		expectedStatus int
	}{
		{
			name: "valid daily activity",
			body: dto.CreateActivityRequest{
				Title:            "Morning Run",
				Description:      "30 min jog",
				CompletionAmount: 1,
				Frequency:        "daily",
				DayTime:          "morning",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "Morning Run", data["title"])
				assert.Equal(t, "30 min jog", data["description"])
			},
		},
		{
			name: "valid weekly activity",
			body: dto.CreateActivityRequest{
				Title:            "Gym",
				CompletionAmount: 1,
				Frequency:        "weekly",
				DayFrequency:     `["monday","friday"]`,
				DayTime:          "evening",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "Gym", data["title"])
			},
		},
		{
			name:           "invalid request body",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid frequency",
			body: dto.CreateActivityRequest{
				Title:     "Invalid",
				Frequency: "weekly",
				DayTime:   "morning",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/activities", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil {
				var resp map[string]interface{}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetUserActivitiesHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	_, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Test Activity 1",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	_, err = svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Test Activity 2",
		CompletionAmount: 1,
		Frequency:        "weekly",
		DayFrequency:     `["monday"]`,
		DayTime:          "evening",
	})
	require.NoError(t, err)

	t.Run("get all active activities", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/activities", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(2), resp["count"])

		data := resp["data"].([]interface{})
		assert.Len(t, data, 2)
	})

	t.Run("get activities with include_inactive", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/activities?include_inactive=true", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestGetTodayActivitiesHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	_, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Daily Task",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	t.Run("get today's activities", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/activities/today", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.GreaterOrEqual(t, int(resp["count"].(float64)), 1)
	})
}

func TestGetActivityHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	activity, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Get Test",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	t.Run("get existing activity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/activities/%d", activity.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "Get Test", data["title"])
	})

	t.Run("get non-existent activity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/activities/9999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid activity id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/activities/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUpdateActivityHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	activity, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Update Test",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	t.Run("update title", func(t *testing.T) {
		newTitle := "Updated Title"
		updateReq := dto.UpdateActivityRequest{
			Title: &newTitle,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/activities/%d", activity.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "Updated Title", data["title"])
	})

	t.Run("deactivate activity", func(t *testing.T) {
		isActive := false
		updateReq := dto.UpdateActivityRequest{
			IsActive: &isActive,
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/activities/%d", activity.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestDeleteActivityHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	activity, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Delete Test",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	t.Run("delete successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/activities/%d", activity.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp["message"], "deleted successfully")
	})

	t.Run("delete non-existent activity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/activities/9999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRecordActivityHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	activity, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Record Test",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	t.Run("record successfully", func(t *testing.T) {
		recordReq := dto.RecordActivityRequest{
			Notes:          "Completed today!",
			CompletionDate: time.Now(),
		}
		body, _ := json.Marshal(recordReq)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/activities/%d/record", activity.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, "Completed today!", data["notes"])
	})

	t.Run("record with empty body", func(t *testing.T) {
		recordReq := dto.RecordActivityRequest{}
		body, _ := json.Marshal(recordReq)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/activities/%d/record", activity.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestGetActivityRecordsHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	activity, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Records Test",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err = svc.RecordActivity(1, activity.ID, &dto.RecordActivityRequest{
			Notes: fmt.Sprintf("Completion %d", i+1),
		})
		require.NoError(t, err)
	}

	t.Run("get all records", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/activities/%d/records", activity.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(3), resp["count"])
	})

	t.Run("get records with limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/activities/%d/records?limit=2", activity.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(2), resp["count"])
	})
}

func TestGetActivityStatsHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	activity, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Stats Test",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		_, err = svc.RecordActivity(1, activity.ID, &dto.RecordActivityRequest{})
		require.NoError(t, err)
	}

	t.Run("get stats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/activities/%d/stats", activity.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp["data"].(map[string]interface{})
		assert.Equal(t, float64(5), data["totalCompletions"])
	})
}

func TestRevertLastCompletionHandler(t *testing.T) {
	router, svc, cleanup := setupTestRouter(t)
	defer cleanup()

	activity, err := svc.CreateActivity(1, &dto.CreateActivityRequest{
		Title:            "Revert Test",
		CompletionAmount: 1,
		Frequency:        "daily",
		DayTime:          "morning",
	})
	require.NoError(t, err)

	_, err = svc.RecordActivity(1, activity.ID, &dto.RecordActivityRequest{
		Notes: "To be reverted",
	})
	require.NoError(t, err)

	t.Run("revert successfully", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/activities/%d/revert", activity.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp["message"], "reverted")
	})

	t.Run("revert with no completions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/activities/%d/revert", activity.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUnauthorizedAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	controller := NewActivityController(nil)
	activities := router.Group("/activities")
	{
		activities.POST("", controller.CreateActivity)
	}

	t.Run("missing user context", func(t *testing.T) {
		body, _ := json.Marshal(dto.CreateActivityRequest{
			Title:     "Test",
			Frequency: "daily",
			DayTime:   "morning",
		})

		req := httptest.NewRequest(http.MethodPost, "/activities", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
