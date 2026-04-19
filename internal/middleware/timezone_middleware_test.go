package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Subject: internal/middleware/timezone_middleware.go — TimezoneMiddleware
// Scope:
//   - Extracts timezone from X-Timezone header
//   - Parses timezone into *time.Location
//   - Stores location in gin context
//   - Provides GetTimezoneFromContext helper function
// Out of scope:
//   - Timezone-based calculations in business logic
//   - Client-side timezone detection
// Setup: Tests use real timezone database (IANA tz database).

// Extracts a valid timezone from the X-Timezone header and stores it in context.
// Covers common IANA timezone identifiers like America/New_York.
func TestTimezoneMiddleware_ValidTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("X-Timezone", "America/New_York")

	middleware := TimezoneMiddleware()
	middleware(c)

	loc, exists := c.Get(TimezoneContextKey)
	assert.True(t, exists)
	assert.NotNil(t, loc)

	location, ok := loc.(*time.Location)
	assert.True(t, ok)
	assert.Equal(t, "America/New_York", location.String())
}

// Defaults to UTC when the X-Timezone header is missing.
// This ensures consistent behavior for clients that don't send timezone info.
func TestTimezoneMiddleware_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	// No X-Timezone header

	middleware := TimezoneMiddleware()
	middleware(c)

	loc, exists := c.Get(TimezoneContextKey)
	assert.True(t, exists)

	location, ok := loc.(*time.Location)
	assert.True(t, ok)
	assert.Equal(t, time.UTC, location)
}

// Defaults to UTC when an invalid timezone identifier is provided.
// Invalid timezones should not cause errors but should fall back to UTC.
func TestTimezoneMiddleware_InvalidTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("X-Timezone", "NotARealTimezone")

	middleware := TimezoneMiddleware()
	middleware(c)

	loc, exists := c.Get(TimezoneContextKey)
	assert.True(t, exists)

	location, ok := loc.(*time.Location)
	assert.True(t, ok)
	assert.Equal(t, time.UTC, location)
}

// Handles empty X-Timezone header value the same as missing header.
// An empty string should fall back to UTC.
func TestTimezoneMiddleware_EmptyHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("X-Timezone", "")

	middleware := TimezoneMiddleware()
	middleware(c)

	loc, exists := c.Get(TimezoneContextKey)
	assert.True(t, exists)

	location, ok := loc.(*time.Location)
	assert.True(t, ok)
	assert.Equal(t, time.UTC, location)
}

// Continues to next handler after processing timezone.
// The middleware should not abort the request chain.
func TestTimezoneMiddleware_ContinuesToNext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("X-Timezone", "Europe/London")

	middleware := TimezoneMiddleware()
	middleware(c)

	assert.False(t, c.IsAborted())
}

// Retrieves timezone from context using GetTimezoneFromContext helper.
// This is the primary way handlers should access the timezone.
func TestGetTimezoneFromContext_WithTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Set(TimezoneContextKey, time.FixedZone("TestZone", 5*60*60))

	loc := GetTimezoneFromContext(c)
	assert.NotNil(t, loc)
	assert.Equal(t, time.FixedZone("TestZone", 5*60*60).String(), loc.String())
}

// Returns UTC when no timezone is stored in context.
// This is the fallback for GetTimezoneFromContext.
func TestGetTimezoneFromContext_NoTimezone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	// No timezone set in context

	loc := GetTimezoneFromContext(c)
	assert.NotNil(t, loc)
	assert.Equal(t, time.UTC, loc)
}

// Returns UTC when context contains non-timezone value.
// Handles corrupted context values gracefully.
func TestGetTimezoneFromContext_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Set(TimezoneContextKey, "not-a-location") // Wrong type

	loc := GetTimezoneFromContext(c)
	assert.NotNil(t, loc)
	assert.Equal(t, time.UTC, loc)
}

// Handles various valid IANA timezone identifiers.
// Uses table-driven tests for common timezones.
func TestTimezoneMiddleware_VariousTimezones(t *testing.T) {
	testCases := []struct {
		name     string
		timezone string
	}{
		{"UTC", "UTC"},
		{"GMT", "GMT"},
		{"EST_NewYork", "America/New_York"},
		{"PST_LA", "America/Los_Angeles"},
		{"Chicago", "America/Chicago"},
		{"London", "Europe/London"},
		{"Paris", "Europe/Paris"},
		{"Berlin", "Europe/Berlin"},
		{"Tokyo", "Asia/Tokyo"},
		{"Sydney", "Australia/Sydney"},
		{"Shanghai", "Asia/Shanghai"},
		{"Dubai", "Asia/Dubai"},
		{"Mumbai", "Asia/Kolkata"},
		{"SaoPaulo", "America/Sao_Paulo"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("GET", "/api/test", nil)
			c.Request.Header.Set("X-Timezone", tc.timezone)

			middleware := TimezoneMiddleware()
			middleware(c)

			loc := GetTimezoneFromContext(c)
			assert.NotNil(t, loc)
			assert.Equal(t, tc.timezone, loc.String())
		})
	}
}

// Handles timezone identifiers with invalid format gracefully.
// Tests various malformed timezone strings.
func TestTimezoneMiddleware_InvalidTimezoneFormats(t *testing.T) {
	invalidTimezones := []string{
		"NotARealTimezone",
		"America/",
		"/New_York",
		"UTC+5",
		"GMT-5",
		"invalid/with/slashes",
		"123",
		"",
	}

	for _, tz := range invalidTimezones {
		t.Run(tz, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("GET", "/api/test", nil)
			c.Request.Header.Set("X-Timezone", tz)

			middleware := TimezoneMiddleware()
			middleware(c)

			// Should not panic and should default to UTC
			loc := GetTimezoneFromContext(c)
			assert.NotNil(t, loc)
			assert.Equal(t, time.UTC, loc)
		})
	}
}

// Full integration: middleware sets timezone and handler retrieves it.
// Simulates real usage flow from HTTP request to handler.
func TestTimezoneMiddleware_EndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(TimezoneMiddleware())

	var capturedLocation *time.Location
	engine.GET("/api/time", func(c *gin.Context) {
		capturedLocation = GetTimezoneFromContext(c)
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/time", nil)
	req.Header.Set("X-Timezone", "Pacific/Auckland")

	engine.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotNil(t, capturedLocation)
	assert.Equal(t, "Pacific/Auckland", capturedLocation.String())
}
