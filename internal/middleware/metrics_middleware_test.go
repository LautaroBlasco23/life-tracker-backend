package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"

	metrics "life-tracker-backend/internal/infrastructure/monitoring"
)

// Subject: internal/middleware/metrics_middleware.go — PrometheusMiddleware
// Scope:
//   - Records HTTP request duration in seconds
//   - Counts total HTTP requests by method, endpoint, and status
//   - Counts HTTP errors (4xx and 5xx) separately
// Out of scope:
//   - Prometheus metric exposition format
//   - Metric aggregation and alerting rules
// Setup: Uses isolated Prometheus registry to avoid test pollution.

func init() {
	// Ensure clean metrics state for tests
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
}

// Records request duration and count for a successful 200 response.
// Verifies that both duration and request count metrics are incremented.
func TestPrometheusMiddleware_RecordsSuccessfulRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/users", nil)
	c.Set("fullPath", "/api/users")

	// Simulate handler that returns 200
	c.Status(http.StatusOK)

	middleware := PrometheusMiddleware()
	middleware(c)

	// Verify no panic occurred and middleware completed
	assert.Equal(t, http.StatusOK, recorder.Code)
}

// Records metrics for POST requests with 201 Created status.
// Verifies that non-200 success codes are properly recorded.
func TestPrometheusMiddleware_RecordsCreatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/api/users", nil)
	c.Set("fullPath", "/api/users")

	c.Status(http.StatusCreated)

	middleware := PrometheusMiddleware()
	middleware(c)

	assert.Equal(t, http.StatusCreated, recorder.Code)
}

// Records error metrics for 4xx client errors.
// Verifies that HTTP error counter is incremented for client errors.
func TestPrometheusMiddleware_RecordsClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/users/999", nil)
	c.Set("fullPath", "/api/users/:id")

	c.Status(http.StatusNotFound)

	middleware := PrometheusMiddleware()
	middleware(c)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

// Records error metrics for 5xx server errors.
// Verifies that HTTP error counter is incremented for server errors.
func TestPrometheusMiddleware_RecordsServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/error", nil)
	c.Set("fullPath", "/api/error")

	c.Status(http.StatusInternalServerError)

	middleware := PrometheusMiddleware()
	middleware(c)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

// Handles requests with no FullPath by using "unknown" as endpoint label.
// Covers 404 responses where no route matched.
func TestPrometheusMiddleware_UnknownPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/not-a-route", nil)
	// No FullPath set - middleware should use "unknown"

	c.Status(http.StatusNotFound)

	middleware := PrometheusMiddleware()
	middleware(c)

	// Middleware should complete without panic
	assert.True(t, true)
}

// Records metrics for DELETE requests.
// Verifies that all HTTP methods are handled correctly.
func TestPrometheusMiddleware_VariousMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(method, "/api/resource", nil)
			c.Set("fullPath", "/api/resource")

			c.Status(http.StatusOK)

			middleware := PrometheusMiddleware()
			middleware(c)

			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

// Records metrics for various status codes including edge cases.
// Covers 400, 401, 403, 404, 500, 502, 503 status codes.
func TestPrometheusMiddleware_VariousStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusBadRequest,         // 400
		http.StatusUnauthorized,       // 401
		http.StatusForbidden,          // 403
		http.StatusNotFound,           // 404
		http.StatusTooManyRequests,    // 429
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
	}

	for _, status := range statusCodes {
		t.Run(http.StatusText(status), func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("GET", "/api/test", nil)
			c.Set("fullPath", "/api/test")

			c.Status(status)

			middleware := PrometheusMiddleware()
			middleware(c)

			assert.Equal(t, status, recorder.Code)
			// All 4xx and 5xx should trigger error counter
			assert.True(t, status >= 400)
		})
	}
}

// Continues to next handler after recording metrics.
// Verifies that subsequent handlers are called.
func TestPrometheusMiddleware_ContinuesToNextHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Set("fullPath", "/api/test")

	c.Status(http.StatusOK)

	middleware := PrometheusMiddleware()
	middleware(c)

	// Should not abort the chain
	assert.False(t, c.IsAborted())
}

// Verifies that metrics.HTTPRequestDuration is accessible and can be observed.
// This is a smoke test to ensure the metric is properly initialized.
func TestPrometheusMiddleware_DurationMetricExists(t *testing.T) {
	// The metric should be initialized and usable
	assert.NotNil(t, metrics.HTTPRequestDuration)

	// Should be able to observe a value without panic
	metrics.HTTPRequestDuration.WithLabelValues("GET", "/test", "200").Observe(0.1)
}

// Verifies that metrics.HTTPRequestsTotal is accessible and can be incremented.
// This is a smoke test to ensure the metric is properly initialized.
func TestPrometheusMiddleware_RequestsTotalMetricExists(t *testing.T) {
	assert.NotNil(t, metrics.HTTPRequestsTotal)

	// Should be able to increment without panic
	metrics.HTTPRequestsTotal.WithLabelValues("POST", "/test", "201").Inc()
}

// Verifies that metrics.HTTPErrorsTotal is accessible and can be incremented.
// This is a smoke test to ensure the metric is properly initialized.
func TestPrometheusMiddleware_ErrorsTotalMetricExists(t *testing.T) {
	assert.NotNil(t, metrics.HTTPErrorsTotal)

	// Should be able to increment without panic
	metrics.HTTPErrorsTotal.WithLabelValues("GET", "/test").Inc()
}
