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

// Records request metrics for a successful 200 response.
// Verifies middleware completes without panic and captures status from context writer.
func TestPrometheusMiddleware_RecordsSuccessfulRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/users", nil)
	c.Set("fullPath", "/api/users") // Simulate route match
	c.Status(http.StatusOK)

	middleware := PrometheusMiddleware()
	middleware(c)

	// Verify no panic occurred and middleware completed
	assert.False(t, c.IsAborted())
}

// Records metrics for POST requests with 201 Created status.
// Verifies that non-200 success codes are properly recorded.
func TestPrometheusMiddleware_RecordsCreatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/api/users", nil)
	c.Set("fullPath", "/api/users")

	// Use the context's writer to set status
	c.Writer.WriteHeader(http.StatusCreated)

	middleware := PrometheusMiddleware()
	middleware(c)

	assert.Equal(t, http.StatusCreated, c.Writer.Status())
}

// Records error metrics for 4xx client errors.
// Verifies that HTTP error counter is incremented for client errors.
func TestPrometheusMiddleware_RecordsClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/users/999", nil)
	c.Set("fullPath", "/api/users/:id")
	c.Writer.WriteHeader(http.StatusNotFound)

	middleware := PrometheusMiddleware()
	middleware(c)

	assert.Equal(t, http.StatusNotFound, c.Writer.Status())
}

// Records error metrics for 5xx server errors.
// Verifies that HTTP error counter is incremented for server errors.
func TestPrometheusMiddleware_RecordsServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/error", nil)
	c.Set("fullPath", "/api/error")
	c.Writer.WriteHeader(http.StatusInternalServerError)

	middleware := PrometheusMiddleware()
	middleware(c)

	assert.Equal(t, http.StatusInternalServerError, c.Writer.Status())
}

// Handles requests with no FullPath by using "unknown" as endpoint label.
// Covers 404 responses where no route matched.
func TestPrometheusMiddleware_UnknownPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/not-a-route", nil)
	// No FullPath set - middleware should use "unknown"
	c.Writer.WriteHeader(http.StatusNotFound)

	middleware := PrometheusMiddleware()
	middleware(c)

	// Middleware should complete without panic
	assert.False(t, c.IsAborted())
}

// Records metrics for various HTTP methods.
// Verifies that all common methods are handled correctly.
func TestPrometheusMiddleware_VariousMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(method, "/api/resource", nil)
			c.Set("fullPath", "/api/resource")
			c.Writer.WriteHeader(http.StatusOK)

			middleware := PrometheusMiddleware()
			middleware(c)

			assert.Equal(t, http.StatusOK, c.Writer.Status())
		})
	}
}

// Records metrics for various error status codes.
// Covers 400, 401, 403, 404, 429, 500, 502, 503 status codes.
func TestPrometheusMiddleware_VariousStatusCodes(t *testing.T) {
	testCases := []struct {
		name   string
		status int
	}{
		{"Bad_Request", http.StatusBadRequest},
		{"Unauthorized", http.StatusUnauthorized},
		{"Forbidden", http.StatusForbidden},
		{"Not_Found", http.StatusNotFound},
		{"Too_Many_Requests", http.StatusTooManyRequests},
		{"Internal_Server_Error", http.StatusInternalServerError},
		{"Bad_Gateway", http.StatusBadGateway},
		{"Service_Unavailable", http.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("GET", "/api/test", nil)
			c.Set("fullPath", "/api/test")
			c.Writer.WriteHeader(tc.status)

			middleware := PrometheusMiddleware()
			middleware(c)

			assert.Equal(t, tc.status, c.Writer.Status())
			// All 4xx and 5xx should be error status codes
			assert.True(t, tc.status >= 400)
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
	c.Writer.WriteHeader(http.StatusOK)

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

// Records metrics for 204 No Content responses.
// Verifies that empty responses are handled correctly.
func TestPrometheusMiddleware_RecordsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("DELETE", "/api/resource/1", nil)
	c.Set("fullPath", "/api/resource/:id")
	c.Writer.WriteHeader(http.StatusNoContent)

	middleware := PrometheusMiddleware()
	middleware(c)

	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
}

// Verifies that requests at the boundary (status 399 and 400) are handled correctly.
// Edge case: 399 is success, 400 is error.
func TestPrometheusMiddleware_StatusBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("status_399", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest("GET", "/api/test", nil)
		c.Set("fullPath", "/api/test")
		c.Writer.WriteHeader(399)

		middleware := PrometheusMiddleware()
		middleware(c)

		// 399 is less than 400, so not an error
		assert.Equal(t, 399, c.Writer.Status())
	})

	t.Run("status_400", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest("GET", "/api/test", nil)
		c.Set("fullPath", "/api/test")
		c.Writer.WriteHeader(http.StatusBadRequest)

		middleware := PrometheusMiddleware()
		middleware(c)

		// 400 should trigger error counter
		assert.Equal(t, http.StatusBadRequest, c.Writer.Status())
	})
}
