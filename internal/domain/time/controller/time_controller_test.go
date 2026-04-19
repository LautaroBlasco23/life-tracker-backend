package controller

// time_controller_test.go
//
// Subject: internal/domain/time/controller/time_controller.go
// Scope:   HTTP handler behavior for time record CRUD operations and statistics
//          Tests from the caller's (HTTP client) perspective — validates request
//          parsing, response formatting, and status codes.
// Out of scope:
//   - Service layer business logic → time_service_test.go
//   - Repository/database behavior → time_repository_test.go
//   - Route/middleware configuration → routes tests
// Setup:   Connects to real PostgreSQL via ENVIRONMENT=test config; creates
//          test records in a transaction-per-test pattern with table cleanup.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/time/dto"
	"life-tracker-backend/internal/domain/time/model"
	"life-tracker-backend/internal/domain/time/service"

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

// setupTestRouter creates a test Gin router with the time controller
// and a real database connection, returning a cleanup function.
func setupTestRouter(t *testing.T) (*gin.Engine, *service.TimeService, func()) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to PostgreSQL")

	db.Exec("DROP TABLE IF EXISTS time_records CASCADE")
	require.NoError(t, db.AutoMigrate(&model.TimeRecord{}))

	timeService := service.NewTimeService(db)
	controller := NewTimeController(timeService)

	router := gin.New()

	// Mock auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("email", "test@example.com")
		c.Set("timezone", "America/New_York")
		c.Next()
	})

	// Register routes
	router.POST("/time", controller.CreateRecord)
	router.GET("/time", controller.GetRecords)
	router.GET("/time/stats", controller.GetStats)
	router.GET("/time/:id", controller.GetRecord)
	router.PUT("/time/:id", controller.UpdateRecord)
	router.DELETE("/time/:id", controller.DeleteRecord)

	cleanup := func() {
		db.Exec("DROP TABLE IF EXISTS time_records CASCADE")
	}

	return router, timeService, cleanup
}

// createTestRecord creates a time record directly in the database for testing.
func createTestRecord(t *testing.T, db *gorm.DB, userID uint, category string, duration int, createdAt time.Time) *model.TimeRecord {
	record := &model.TimeRecord{
		UserID:          userID,
		Category:        category,
		Description:     "Test description for " + category,
		DurationMinutes: duration,
		CreatedAt:       createdAt,
	}
	err := db.Create(record).Error
	require.NoError(t, err)
	return record
}

// createFreshDB creates a fresh database connection for sub-tests that need
// to set up their own data independently.
func createFreshDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// CreateRecord tests

// Creates a time record with valid data and verifies the response contains
// the expected fields with a 201 Created status.
func TestCreateRecord_WithValidRequest_CreatesRecord(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	reqBody := dto.CreateTimeRecordRequest{
		Category:        "Work",
		Description:     "Completed project documentation",
		DurationMinutes: 120,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/time", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Time record created successfully", resp["message"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Work", data["category"])
	assert.Equal(t, "Completed project documentation", data["description"])
	assert.Equal(t, float64(120), data["durationMinutes"])
	assert.NotZero(t, data["id"])
}

// Rejects a request with malformed JSON body and returns a 400 Bad Request
// with error details in the response.
func TestCreateRecord_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/time", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Invalid request")
	assert.NotNil(t, resp["details"])
}

// Rejects a request missing the required category field with a 400 error.
func TestCreateRecord_WithMissingCategory_ReturnsValidationError(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	reqBody := map[string]interface{}{
		"description":     "Missing category",
		"durationMinutes": 60,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/time", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Invalid request")
}

// Rejects a request with empty category (violates min=1 validation).
func TestCreateRecord_WithEmptyCategory_ReturnsValidationError(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	reqBody := dto.CreateTimeRecordRequest{
		Category:        "",
		Description:     "Empty category test",
		DurationMinutes: 30,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/time", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Rejects a request with zero duration (violates min=1 validation).
func TestCreateRecord_WithZeroDuration_ReturnsValidationError(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	reqBody := dto.CreateTimeRecordRequest{
		Category:        "Test",
		Description:     "Zero duration test",
		DurationMinutes: 0,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/time", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// GetRecords tests

// Returns an empty list when the user has no time records.
// Edge case: ensures the response structure is correct even with zero records.
func TestGetRecords_WithNoRecords_ReturnsEmptyList(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/time", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Time records retrieved successfully", resp["message"])
	assert.Equal(t, float64(0), resp["count"])
	assert.Empty(t, resp["data"])
}

// Returns all time records for the authenticated user when no filters are applied.
func TestGetRecords_WithNoFilters_ReturnsAllRecords(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	now := time.Now()
	createTestRecord(t, db, 1, "Work", 60, now)
	createTestRecord(t, db, 1, "Study", 45, now.Add(-time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/time", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["count"])

	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
}

// Filters records by category query parameter, returning only matching records.
func TestGetRecords_WithCategoryFilter_ReturnsFilteredRecords(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	now := time.Now()
	createTestRecord(t, db, 1, "Work", 60, now)
	createTestRecord(t, db, 1, "Personal", 30, now)
	createTestRecord(t, db, 1, "Work", 90, now.Add(-time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/time?category=Work", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["count"])

	data := resp["data"].([]interface{})
	for _, record := range data {
		r := record.(map[string]interface{})
		assert.Equal(t, "Work", r["category"])
	}
}

// Filters records by month and year parameters.
func TestGetRecords_WithMonthYearFilter_ReturnsFilteredRecords(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	// Create record in January 2024
	createTestRecord(t, db, 1, "Work", 60, time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))
	// Create record in February 2024
	createTestRecord(t, db, 1, "Work", 90, time.Date(2024, 2, 10, 10, 0, 0, 0, time.UTC))

	req := httptest.NewRequest(http.MethodGet, "/time?month=1&year=2024", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["count"])

	data := resp["data"].([]interface{})
	assert.Len(t, data, 1)
}

// Filters records by date range (start_date and end_date parameters).
func TestGetRecords_WithDateRangeFilter_ReturnsFilteredRecords(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	createTestRecord(t, db, 1, "Work", 60, time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC))
	createTestRecord(t, db, 1, "Work", 90, time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC))
	createTestRecord(t, db, 1, "Work", 30, time.Date(2024, 4, 1, 10, 0, 0, 0, time.UTC))

	req := httptest.NewRequest(http.MethodGet, "/time?start_date=2024-03-01&end_date=2024-03-31", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["count"])
}

// Ignores invalid month values outside the 1-12 range.
func TestGetRecords_WithInvalidMonth_IgnoresFilter(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	createTestRecord(t, db, 1, "Work", 60, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/time?month=13", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still succeed but ignore the invalid filter
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["count"])
}

// Ignores invalid year values outside the 2000-2100 range.
func TestGetRecords_WithInvalidYear_IgnoresFilter(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	createTestRecord(t, db, 1, "Work", 60, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/time?year=1800", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still succeed but ignore the invalid filter
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["count"])
}

// GetRecord tests

// Retrieves a specific time record by ID with full details.
func TestGetRecord_WithExistingID_ReturnsRecord(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	record := createTestRecord(t, db, 1, "Study", 120, time.Now())

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/time/%d", record.ID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Time record retrieved successfully", resp["message"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(record.ID), data["id"])
	assert.Equal(t, "Study", data["category"])
	assert.Equal(t, "Test description for Study", data["description"])
	assert.Equal(t, float64(120), data["durationMinutes"])
}

// Returns 404 Not Found when the record does not exist.
func TestGetRecord_WithNonExistentID_ReturnsNotFound(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/time/99999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "not found")
}

// Returns 400 Bad Request when the ID parameter is not a valid number.
func TestGetRecord_WithInvalidIDFormat_ReturnsBadRequest(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/time/invalid-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Invalid record ID")
}

// UpdateRecord tests

// Updates all fields of a time record with valid data.
func TestUpdateRecord_WithValidData_UpdatesAllFields(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	record := createTestRecord(t, db, 1, "OldCategory", 60, time.Now())

	newCategory := "UpdatedCategory"
	newDesc := "Updated description"
	newDuration := 90
	updateReq := dto.UpdateTimeRecordRequest{
		Category:        &newCategory,
		Description:     &newDesc,
		DurationMinutes: &newDuration,
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/time/%d", record.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Time record updated successfully", resp["message"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "UpdatedCategory", data["category"])
	assert.Equal(t, "Updated description", data["description"])
	assert.Equal(t, float64(90), data["durationMinutes"])
}

// Updates only the category field, leaving other fields unchanged.
func TestUpdateRecord_WithPartialData_UpdatesOnlyProvidedFields(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	record := createTestRecord(t, db, 1, "OriginalCategory", 60, time.Now())

	newCategory := "PartiallyUpdated"
	updateReq := dto.UpdateTimeRecordRequest{
		Category: &newCategory,
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/time/%d", record.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "PartiallyUpdated", data["category"])
	// Original values preserved
	assert.Equal(t, "Test description for OriginalCategory", data["description"])
	assert.Equal(t, float64(60), data["durationMinutes"])
}

// Returns 404 when trying to update a non-existent record.
func TestUpdateRecord_WithNonExistentID_ReturnsNotFound(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	newCategory := "Updated"
	updateReq := dto.UpdateTimeRecordRequest{
		Category: &newCategory,
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/time/99999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "not found")
}

// Returns 400 when the request body contains invalid JSON.
func TestUpdateRecord_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	record := createTestRecord(t, db, 1, "Test", 60, time.Now())

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/time/%d", record.ID), bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Invalid request")
}

// Returns 400 when the ID parameter is not a valid number.
func TestUpdateRecord_WithInvalidIDFormat_ReturnsBadRequest(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	newCategory := "Updated"
	updateReq := dto.UpdateTimeRecordRequest{
		Category: &newCategory,
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest(http.MethodPut, "/time/invalid", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Invalid record ID")
}

// DeleteRecord tests

// Successfully deletes an existing time record and returns a success message.
func TestDeleteRecord_WithExistingID_DeletesRecord(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	record := createTestRecord(t, db, 1, "ToDelete", 60, time.Now())

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/time/%d", record.ID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Time record deleted successfully", resp["message"])

	// Verify it's actually deleted
	var count int64
	db.Model(&model.TimeRecord{}).Where("id = ?", record.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// Returns 400 when trying to delete a non-existent record.
func TestDeleteRecord_WithNonExistentID_ReturnsError(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/time/99999", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "not found")
}

// Returns 400 when the ID parameter is not a valid number.
func TestDeleteRecord_WithInvalidIDFormat_ReturnsBadRequest(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodDelete, "/time/invalid-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Invalid record ID")
}

// GetStats tests

// Returns correct statistics for user's time records including total minutes,
// record count, and category breakdown.
func TestGetStats_WithRecords_ReturnsCorrectStats(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	db := createFreshDB(t)
	createTestRecord(t, db, 1, "Work", 60, time.Now())
	createTestRecord(t, db, 1, "Work", 120, time.Now())
	createTestRecord(t, db, 1, "Personal", 30, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/time/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Time stats retrieved successfully", resp["message"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(210), data["totalMinutes"])
	assert.Equal(t, float64(3), data["recordCount"])
	assert.Equal(t, "Work", data["topCategory"])
	assert.Equal(t, float64(180), data["topCategoryMinutes"])

	categoryTotals := data["categoryTotals"].(map[string]interface{})
	assert.Equal(t, float64(180), categoryTotals["Work"])
	assert.Equal(t, float64(30), categoryTotals["Personal"])
}

// Returns zero-value stats when user has no records.
func TestGetStats_WithNoRecords_ReturnsEmptyStats(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/time/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["totalMinutes"])
	assert.Equal(t, float64(0), data["recordCount"])
	assert.Equal(t, "", data["topCategory"])
}

// Unauthorized access tests

// Returns 401 when the user context is missing from the request.
func TestCreateRecord_WithoutUserContext_ReturnsUnauthorized(t *testing.T) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	timeService := service.NewTimeService(db)
	controller := NewTimeController(timeService)

	// Router without auth middleware
	router := gin.New()
	router.POST("/time", controller.CreateRecord)

	reqBody := dto.CreateTimeRecordRequest{
		Category:        "Work",
		Description:     "Test",
		DurationMinutes: 60,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/time", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "user ID")
}

// Returns 401 when the user context has invalid type.
func TestGetRecords_WithInvalidUserIDType_ReturnsUnauthorized(t *testing.T) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	timeService := service.NewTimeService(db)
	controller := NewTimeController(timeService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", "not-a-number") // Wrong type
		c.Next()
	})
	router.GET("/time", controller.GetRecords)

	req := httptest.NewRequest(http.MethodGet, "/time", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Table-driven test for edge cases in CreateRecord validation
func TestCreateRecord_ValidationEdgeCases(t *testing.T) {
	router, _, cleanup := setupTestRouter(t)
	defer cleanup()

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "missing category",
			body: map[string]interface{}{
				"description":     "desc",
				"durationMinutes": 60,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing description",
			body: map[string]interface{}{
				"category":        "Work",
				"durationMinutes": 60,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing duration",
			body: map[string]interface{}{
				"category":    "Work",
				"description": "desc",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "negative duration",
			body: map[string]interface{}{
				"category":        "Work",
				"description":     "desc",
				"durationMinutes": -10,
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "category too long",
			body: map[string]interface{}{
				"category":        string(make([]byte, 101)),
				"description":     "desc",
				"durationMinutes": 60,
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/time", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

// Tests that users cannot access other users' records (isolation test).
// Edge case: security boundary — verifies user_id filtering works correctly.
func TestGetRecord_UserIsolation_PreventsAccessToOtherUserRecords(t *testing.T) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS time_records CASCADE")
	require.NoError(t, db.AutoMigrate(&model.TimeRecord{}))

	// Create record for user 2
	record := createTestRecord(t, db, 2, "Secret", 60, time.Now())

	timeService := service.NewTimeService(db)
	controller := NewTimeController(timeService)

	// Router for user 1
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1)) // Different user
		c.Next()
	})
	router.GET("/time/:id", controller.GetRecord)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/time/%d", record.ID), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// User 1 should not see user 2's record
	assert.Equal(t, http.StatusNotFound, w.Code)
}
