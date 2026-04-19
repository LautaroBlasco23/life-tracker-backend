package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// Subject: internal/middleware/auth_middleware.go — JWTAuthMiddleware
// Scope:
//   - Validates JWT tokens from Authorization header
//   - Rejects requests with missing, malformed, or invalid tokens
//   - Extracts user claims and sets them in gin context
// Out of scope:
//   - Token generation logic → internal/domain/auth/service/auth_service_test.go
//   - Database authentication → repository layer tests
// Setup: Uses fixed test secret; tokens created with jwt library.

const testJWTSecret = "test-secret-key-for-middleware-tests"

// generateTestToken creates a JWT token with specified claims for testing.
func generateTestToken(claims *JWTClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(testJWTSecret))
}

// setupTestRouter creates a gin router with the auth middleware for testing.
func setupTestRouter() (*gin.Engine, *gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	return nil, ctx, recorder
}

// Validates that a request with a valid access token passes authentication
// and has userID and email set in the context.
func TestJWTAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	claims := &JWTClaims{
		UserID: 42,
		Email:  "test@example.com",
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tokenString, err := generateTestToken(claims)
	assert.NoError(t, err)

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.False(t, c.IsAborted())
	assert.Equal(t, uint(42), c.GetUint("userID"))
	assert.Equal(t, "test@example.com", c.GetString("email"))
}

// Rejects requests with an Authorization header but missing the token value.
// Covers the case where header is just "Bearer " without a token.
func TestJWTAuthMiddleware_MissingTokenValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer ")

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid token")
}

// Rejects requests with a completely missing Authorization header.
// This is the most common unauthenticated request scenario.
func TestJWTAuthMiddleware_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Authorization header required")
}

// Rejects requests where the Authorization header format is incorrect.
// Covers cases like missing "Bearer" prefix or wrong number of parts.
func TestJWTAuthMiddleware_MalformedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid authorization header format")
}

// Rejects tokens that have expired according to the exp claim.
// Expired tokens should be treated as completely invalid.
func TestJWTAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	claims := &JWTClaims{
		UserID: 42,
		Email:  "test@example.com",
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	tokenString, err := generateTestToken(claims)
	assert.NoError(t, err)

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid token")
}

// Rejects tokens signed with a different secret key.
// This prevents token forgery from unauthorized issuers.
func TestJWTAuthMiddleware_WrongSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	claims := &JWTClaims{
		UserID: 1,
		Email:  "test@example.com",
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	// Sign with wrong secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("wrong-secret"))
	assert.NoError(t, err)

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid token")
}

// Rejects tokens that are not valid base64 or have invalid structure.
// Covers completely garbled token strings.
func TestJWTAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer not-a-valid-token")

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid token")
}

// Rejects refresh tokens when used as access tokens.
// Refresh tokens should not grant API access even if otherwise valid.
func TestJWTAuthMiddleware_WrongTokenType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	claims := &JWTClaims{
		UserID: 1,
		Email:  "test@example.com",
		Type:   "refresh", // Wrong type
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tokenString, err := generateTestToken(claims)
	assert.NoError(t, err)

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid token type")
}

// Validates that the middleware correctly handles tokens with missing type claim.
// Tokens without a type claim should be rejected as invalid.
func TestJWTAuthMiddleware_MissingTypeClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	claims := &JWTClaims{
		UserID: 1,
		Email:  "test@example.com",
		Type:   "", // Empty type
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tokenString, err := generateTestToken(claims)
	assert.NoError(t, err)

	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)

	middleware := JWTAuthMiddleware(testJWTSecret)
	middleware(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid token type")
}
