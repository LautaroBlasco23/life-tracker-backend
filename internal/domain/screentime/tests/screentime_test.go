package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/screentime/controller"
	"life-tracker-backend/internal/domain/screentime/dto"
	"life-tracker-backend/internal/domain/screentime/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	testMongoDB *mongo.Database
	mongoClient *mongo.Client
	cfg         *config.Config
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		panic("failed to connect to MongoDB: " + err.Error())
	}
	if err := client.Ping(ctx, nil); err != nil {
		panic("MongoDB not available: " + err.Error())
	}

	testMongoDB = client.Database(cfg.MongoDatabase)
	mongoClient = client

	code := m.Run()

	if mongoClient != nil {
		_ = mongoClient.Disconnect(context.Background())
	}

	os.Exit(code)
}

func cleanCollection(t *testing.T) {
	t.Helper()
	_ = testMongoDB.Collection("web_visits").Drop(context.Background())
}

// ---- Service tests ----

func TestRecordVisit(t *testing.T) {
	cleanCollection(t)

	svc := service.NewVisitService(testMongoDB)

	tests := []struct {
		name    string
		userID  uint
		req     *dto.RecordVisitRequest
		wantErr bool
		check   func(t *testing.T, resp *dto.VisitResponse)
	}{
		{
			name:   "valid domain and visitedAt",
			userID: 1,
			req: &dto.RecordVisitRequest{
				Domain:    "github.com",
				VisitedAt: time.Now(),
			},
			wantErr: false,
			check: func(t *testing.T, resp *dto.VisitResponse) {
				assert.Equal(t, "github.com", resp.Domain)
				assert.NotEmpty(t, resp.ID)
			},
		},
		{
			name:   "two visits same domain both stored",
			userID: 2,
			req: &dto.RecordVisitRequest{
				Domain:    "youtube.com",
				VisitedAt: time.Now(),
			},
			wantErr: false,
			check: func(t *testing.T, resp *dto.VisitResponse) {
				assert.Equal(t, "youtube.com", resp.Domain)
				assert.NotEmpty(t, resp.ID)

				resp2, err := svc.RecordVisit(2, &dto.RecordVisitRequest{
					Domain:    "youtube.com",
					VisitedAt: time.Now(),
				})
				require.NoError(t, err)
				assert.NotEqual(t, resp.ID, resp2.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.RecordVisit(tt.userID, tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.check != nil {
					tt.check(t, resp)
				}
			}
		})
	}
}

func TestGetVisits(t *testing.T) {
	cleanCollection(t)

	svc := service.NewVisitService(testMongoDB)
	userID := uint(10)
	otherUserID := uint(11)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// seed visits
	_, err := svc.RecordVisit(userID, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: now})
	require.NoError(t, err)
	_, err = svc.RecordVisit(userID, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: yesterday})
	require.NoError(t, err)
	_, err = svc.RecordVisit(userID, &dto.RecordVisitRequest{Domain: "instagram.com", VisitedAt: twoDaysAgo})
	require.NoError(t, err)
	_, err = svc.RecordVisit(otherUserID, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: now})
	require.NoError(t, err)

	t.Run("no filters returns all visits for user", func(t *testing.T) {
		visits, err := svc.GetVisits(userID, "", nil, nil, 100)
		require.NoError(t, err)
		assert.Len(t, visits, 3)
	})

	t.Run("no other users visits returned", func(t *testing.T) {
		visits, err := svc.GetVisits(userID, "", nil, nil, 100)
		require.NoError(t, err)
		for _, v := range visits {
			// all visits belong to userID — none from otherUserID
			assert.NotEmpty(t, v.ID)
		}
		// otherUserID has 1 visit; total in collection is 4; user gets only 3
		assert.Len(t, visits, 3)
	})

	t.Run("filter by domain", func(t *testing.T) {
		visits, err := svc.GetVisits(userID, "github.com", nil, nil, 100)
		require.NoError(t, err)
		assert.Len(t, visits, 2)
		for _, v := range visits {
			assert.Equal(t, "github.com", v.Domain)
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		from := twoDaysAgo.Add(-time.Hour)
		to := yesterday.Add(-time.Hour)
		visits, err := svc.GetVisits(userID, "", &from, &to, 100)
		require.NoError(t, err)
		assert.Len(t, visits, 1)
		assert.Equal(t, "instagram.com", visits[0].Domain)
	})

	t.Run("limit param respected", func(t *testing.T) {
		visits, err := svc.GetVisits(userID, "", nil, nil, 2)
		require.NoError(t, err)
		assert.Len(t, visits, 2)
	})
}

func TestGetStats(t *testing.T) {
	cleanCollection(t)

	svc := service.NewVisitService(testMongoDB)
	userID := uint(20)
	otherUserID := uint(21)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// seed: github×3, youtube×2, instagram×1
	for i := 0; i < 3; i++ {
		_, err := svc.RecordVisit(userID, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: now})
		require.NoError(t, err)
	}
	for i := 0; i < 2; i++ {
		_, err := svc.RecordVisit(userID, &dto.RecordVisitRequest{Domain: "youtube.com", VisitedAt: yesterday})
		require.NoError(t, err)
	}
	_, err := svc.RecordVisit(userID, &dto.RecordVisitRequest{Domain: "instagram.com", VisitedAt: twoDaysAgo})
	require.NoError(t, err)

	// other user — must not appear
	_, err = svc.RecordVisit(otherUserID, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: now})
	require.NoError(t, err)

	t.Run("returns per-domain counts sorted by count desc", func(t *testing.T) {
		stats, err := svc.GetStats(userID, nil, nil)
		require.NoError(t, err)
		require.Len(t, stats, 3)
		assert.Equal(t, "github.com", stats[0].Domain)
		assert.Equal(t, int64(3), stats[0].Count)
		assert.Equal(t, "youtube.com", stats[1].Domain)
		assert.Equal(t, int64(2), stats[1].Count)
		assert.Equal(t, "instagram.com", stats[2].Domain)
		assert.Equal(t, int64(1), stats[2].Count)
	})

	t.Run("no visits returns empty slice not nil", func(t *testing.T) {
		stats, err := svc.GetStats(uint(999), nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Empty(t, stats)
	})

	t.Run("filter by date range counts only visits in range", func(t *testing.T) {
		from := yesterday.Add(-time.Hour)
		to := yesterday.Add(time.Hour)
		stats, err := svc.GetStats(userID, &from, &to)
		require.NoError(t, err)
		require.Len(t, stats, 1)
		assert.Equal(t, "youtube.com", stats[0].Domain)
		assert.Equal(t, int64(2), stats[0].Count)
	})

	t.Run("no cross-user data leakage", func(t *testing.T) {
		stats, err := svc.GetStats(otherUserID, nil, nil)
		require.NoError(t, err)
		require.Len(t, stats, 1)
		assert.Equal(t, "github.com", stats[0].Domain)
		assert.Equal(t, int64(1), stats[0].Count)
	})
}

// ---- Controller / HTTP handler tests ----

func setupScreenTimeRouter(t *testing.T) (*gin.Engine, *service.VisitService, func()) {
	t.Helper()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	require.NoError(t, err)
	require.NoError(t, client.Ping(context.Background(), nil), "MongoDB not available")

	dbName := "test_screentime_" + primitive.NewObjectID().Hex()
	mongoDB := client.Database(dbName)

	visitService := service.NewVisitService(mongoDB)
	ctrl := controller.NewVisitController(visitService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	screenTime := router.Group("/screen-time")
	{
		screenTime.POST("/visits", ctrl.RecordVisit)
		screenTime.GET("/visits", ctrl.GetVisits)
		screenTime.GET("/stats", ctrl.GetStats)
	}

	cleanup := func() {
		_ = mongoDB.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}

	return router, visitService, cleanup
}

func TestRecordVisitHandler(t *testing.T) {
	router, _, cleanup := setupScreenTimeRouter(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp map[string]interface{})
	}{
		{
			name: "valid body returns 201",
			body: dto.RecordVisitRequest{
				Domain:    "github.com",
				VisitedAt: time.Now(),
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "github.com", data["domain"])
			},
		},
		{
			name: "missing domain returns 400",
			body: map[string]interface{}{
				"visitedAt": time.Now(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing visitedAt returns 400",
			body: map[string]interface{}{
				"domain": "github.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed JSON returns 400",
			body:           "not-valid-json{",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/screen-time/visits", bytes.NewBuffer(body))
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

func TestGetVisitsHandler(t *testing.T) {
	router, svc, cleanup := setupScreenTimeRouter(t)
	defer cleanup()

	now := time.Now()

	// seed 3 visits: 2 github, 1 instagram
	_, err := svc.RecordVisit(1, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: now})
	require.NoError(t, err)
	_, err = svc.RecordVisit(1, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: now.Add(-time.Hour)})
	require.NoError(t, err)
	_, err = svc.RecordVisit(1, &dto.RecordVisitRequest{Domain: "instagram.com", VisitedAt: now.Add(-2 * time.Hour)})
	require.NoError(t, err)

	t.Run("returns all 3 visits", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/screen-time/visits", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(3), resp["count"])
	})

	t.Run("filter by domain returns only matching", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/screen-time/visits?domain=instagram.com", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(1), resp["count"])
		data := resp["data"].([]interface{})
		assert.Len(t, data, 1)
		entry := data[0].(map[string]interface{})
		assert.Equal(t, "instagram.com", entry["domain"])
	})

	t.Run("limit param returns only 2", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/screen-time/visits?limit=2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(2), resp["count"])
	})

	t.Run("from and to date filters work", func(t *testing.T) {
		from := now.Add(-90 * time.Minute).Format("2006-01-02")
		to := now.Add(-30 * time.Minute).Format("2006-01-02")

		// from == to (same day) — all visits on that day qualify
		req := httptest.NewRequest(http.MethodGet, "/screen-time/visits?from="+from+"&to="+to, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		// response must be valid (no error key)
		assert.NotContains(t, resp, "error")
	})
}

func TestGetStatsHandler(t *testing.T) {
	router, svc, cleanup := setupScreenTimeRouter(t)
	defer cleanup()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// seed: github×2, youtube×1
	_, err := svc.RecordVisit(1, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: now})
	require.NoError(t, err)
	_, err = svc.RecordVisit(1, &dto.RecordVisitRequest{Domain: "github.com", VisitedAt: now})
	require.NoError(t, err)
	_, err = svc.RecordVisit(1, &dto.RecordVisitRequest{Domain: "youtube.com", VisitedAt: yesterday})
	require.NoError(t, err)

	t.Run("stats has correct counts", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/screen-time/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, float64(2), resp["count"])
		data := resp["data"].([]interface{})
		require.Len(t, data, 2)
		top := data[0].(map[string]interface{})
		assert.Equal(t, "github.com", top["domain"])
		assert.Equal(t, float64(2), top["count"])
	})

	t.Run("from/to date params work", func(t *testing.T) {
		from := yesterday.Add(-time.Hour).Format("2006-01-02")
		to := yesterday.Add(time.Hour).Format("2006-01-02")
		req := httptest.NewRequest(http.MethodGet, "/screen-time/stats?from="+from+"&to="+to, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotContains(t, resp, "error")
	})
}

func TestGetStatsHandlerEmptyDB(t *testing.T) {
	router, _, cleanup := setupScreenTimeRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/screen-time/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["count"])
	data := resp["data"].([]interface{})
	assert.Empty(t, data)
}

func TestScreenTimeUnauthorized(t *testing.T) {
	// router without userID middleware
	ctrl := controller.NewVisitController(nil)

	router := gin.New()
	screenTime := router.Group("/screen-time")
	{
		screenTime.POST("/visits", ctrl.RecordVisit)
		screenTime.GET("/visits", ctrl.GetVisits)
		screenTime.GET("/stats", ctrl.GetStats)
	}

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/screen-time/visits"},
		{http.MethodGet, "/screen-time/visits"},
		{http.MethodGet, "/screen-time/stats"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path+" without userID returns 401", func(t *testing.T) {
			var reqBody *bytes.Buffer
			if ep.method == http.MethodPost {
				body, _ := json.Marshal(dto.RecordVisitRequest{
					Domain:    "github.com",
					VisitedAt: time.Now(),
				})
				reqBody = bytes.NewBuffer(body)
			} else {
				reqBody = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(ep.method, ep.path, reqBody)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
