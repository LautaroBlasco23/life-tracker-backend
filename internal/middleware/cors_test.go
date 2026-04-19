package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Subject: internal/middleware/cors.go — CORSMiddleware
// Scope:
//   - Sets CORS headers for cross-origin requests
//   - Handles preflight OPTIONS requests
//   - Validates origin against allowed list
// Out of scope:
//   - HTTP routing and endpoint handling
//   - Browser enforcement of CORS policy
// Setup: Uses table-driven tests with configurable allowed origins.

// Allows requests from any origin when configured with wildcard "*".
// The Access-Control-Allow-Origin header should be set to "*".
func TestCORSMiddleware_AllowAllOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://any-origin.com")

	middleware := CORSMiddleware("*")
	middleware(c)

	assert.Equal(t, "*", recorder.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, recorder.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Contains(t, recorder.Header().Get("Access-Control-Allow-Methods"), "POST")
}

// Allows requests from a specific origin that is in the allowed list.
// The response should echo back the request origin.
func TestCORSMiddleware_AllowSpecificOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://app.example.com")

	middleware := CORSMiddleware("https://app.example.com,https://admin.example.com")
	middleware(c)

	assert.Equal(t, "https://app.example.com", recorder.Header().Get("Access-Control-Allow-Origin"))
}

// Rejects requests from origins not in the allowed list.
// The Access-Control-Allow-Origin header should be absent.
func TestCORSMiddleware_DisallowUnknownOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://malicious-site.com")

	middleware := CORSMiddleware("https://app.example.com")
	middleware(c)

	assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
}

// Handles preflight OPTIONS requests by returning 204 No Content.
// The request should be aborted without calling subsequent handlers.
func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("OPTIONS", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://app.example.com")

	handlerCalled := false
	c.Set("handlerCalled", &handlerCalled)

	middleware := CORSMiddleware("https://app.example.com")
	middleware(c)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.True(t, c.IsAborted())
}

// Sets all required CORS headers including allowed methods and headers.
// Covers the full header set required for CORS compliance.
func TestCORSMiddleware_SetsAllHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://app.example.com")

	middleware := CORSMiddleware("https://app.example.com")
	middleware(c)

	assert.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))

	headers := recorder.Header().Get("Access-Control-Allow-Headers")
	assert.Contains(t, headers, "Content-Type")
	assert.Contains(t, headers, "Authorization")
	assert.Contains(t, headers, "X-Timezone")
	assert.Contains(t, headers, "X-CSRF-Token")

	methods := recorder.Header().Get("Access-Control-Allow-Methods")
	assert.Contains(t, methods, "GET")
	assert.Contains(t, methods, "POST")
	assert.Contains(t, methods, "PUT")
	assert.Contains(t, methods, "DELETE")
	assert.Contains(t, methods, "OPTIONS")
}

// Handles requests with no Origin header gracefully.
// Non-CORS requests should still have headers set.
func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	// No Origin header set

	middleware := CORSMiddleware("https://app.example.com")
	middleware(c)

	// Should still set other headers
	assert.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
	assert.NotEmpty(t, recorder.Header().Get("Access-Control-Allow-Headers"))
	assert.NotEmpty(t, recorder.Header().Get("Access-Control-Allow-Methods"))
}

// Handles origins with whitespace in the allowed list correctly.
// The middleware should trim spaces when parsing allowed origins.
func TestCORSMiddleware_OriginsWithWhitespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://second.com")

	middleware := CORSMiddleware("https://first.com, https://second.com , https://third.com")
	middleware(c)

	assert.Equal(t, "https://second.com", recorder.Header().Get("Access-Control-Allow-Origin"))
}

// Continues to next handler for non-OPTIONS requests.
// GET requests should not be aborted.
func TestCORSMiddleware_ContinuesToNextHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://app.example.com")

	middleware := CORSMiddleware("https://app.example.com")
	middleware(c)

	assert.False(t, c.IsAborted())
}

// Validates that empty allowed origins string results in no origin matches.
// An empty string (not "*") should disallow all origins.
func TestCORSMiddleware_EmptyAllowedOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("Origin", "https://any-site.com")

	middleware := CORSMiddleware("")
	middleware(c)

	// Should not allow the origin since nothing is configured
	assert.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	// But should still set other headers
	assert.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
}
