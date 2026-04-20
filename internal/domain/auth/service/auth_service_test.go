package service

import (
	"fmt"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/auth/dto"
	"life-tracker-backend/internal/domain/auth/model"
	userModel "life-tracker-backend/internal/domain/user/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	testDB *gorm.DB
	cfg    *config.Config
)

func TestMain(m *testing.M) {
	os.Setenv("ENVIRONMENT", "test")
	cfg = config.Load()
	code := m.Run()
	os.Exit(code)
}

func setupTestDatabase(t *testing.T) {
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

		require.NoError(t, db.AutoMigrate(&model.Auth{}, &userModel.User{}))
		testDB = db
	}
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	require.NoError(t, testDB.Exec("TRUNCATE TABLE auths, users RESTART IDENTITY CASCADE;").Error)
}

func getTestService(t *testing.T) *AuthService {
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewAuthService(testDB, cfg)
}

func TestAuthService_Register(t *testing.T) {
	service := getTestService(t)

	t.Run("successful registration", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:     "test@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
			Username:  "johndoe",
		}

		tokens, userID, err := service.Register(req)

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.NotZero(t, userID)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, "Bearer", tokens.TokenType)

		var auth model.Auth
		err = testDB.Where("email = ?", req.Email).First(&auth).Error
		assert.NoError(t, err)
		assert.Equal(t, userID, auth.UserID)

		err = bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(req.Password))
		assert.NoError(t, err)

		// Verify username was stored in auth table
		assert.Equal(t, "johndoe", auth.Username)
	})

	t.Run("email already exists", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:     "duplicate@example.com",
			Password:  "password123",
			FirstName: "Jane",
			LastName:  "Doe",
			Username:  "janedoe",
		}

		_, _, err := service.Register(req)
		require.NoError(t, err)

		tokens, userID, err := service.Register(req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Zero(t, userID)
		assert.Contains(t, err.Error(), "user already exists")
	})

	t.Run("username already taken", func(t *testing.T) {
		// First registration with a username
		req1 := &dto.RegisterRequest{
			Email:     "user1@example.com",
			Password:  "password123",
			FirstName: "User",
			LastName:  "One",
			Username:  "uniqueuser",
		}
		_, _, err := service.Register(req1)
		require.NoError(t, err)

		// Second registration with same username but different email
		req2 := &dto.RegisterRequest{
			Email:     "user2@example.com",
			Password:  "password123",
			FirstName: "User",
			LastName:  "Two",
			Username:  "uniqueuser",
		}
		tokens, userID, err := service.Register(req2)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Zero(t, userID)
		assert.Contains(t, err.Error(), "username already taken")
	})

	t.Run("invalid username format", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:     "testuser@example.com",
			Password:  "password123",
			FirstName: "Test",
			LastName:  "User",
			Username:  "Invalid-User!",
		}

		tokens, userID, err := service.Register(req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Zero(t, userID)
		assert.Contains(t, err.Error(), "username must be 3-30 characters: lowercase letters, digits, underscores only")
	})

	t.Run("username too short", func(t *testing.T) {
		req := &dto.RegisterRequest{
			Email:     "testuser@example.com",
			Password:  "password123",
			FirstName: "Test",
			LastName:  "User",
			Username:  "ab",
		}

		tokens, userID, err := service.Register(req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Zero(t, userID)
	})
}

func TestAuthService_Login(t *testing.T) {
	service := getTestService(t)

	registerReq := &dto.RegisterRequest{
		Email:     "login@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Username:  "testuser",
	}
	_, expectedUserID, err := service.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful login with email", func(t *testing.T) {
		req := &dto.LoginRequest{
			Identifier: "login@example.com",
			Password:   "password123",
		}

		tokens, userID, err := service.Login(req)

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.Equal(t, expectedUserID, userID)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("successful login with username", func(t *testing.T) {
		req := &dto.LoginRequest{
			Identifier: "testuser",
			Password:   "password123",
		}

		tokens, userID, err := service.Login(req)

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.Equal(t, expectedUserID, userID)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("successful login with username (case insensitive)", func(t *testing.T) {
		req := &dto.LoginRequest{
			Identifier: "TESTUSER",
			Password:   "password123",
		}

		tokens, userID, err := service.Login(req)

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.Equal(t, expectedUserID, userID)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("user not found by email", func(t *testing.T) {
		req := &dto.LoginRequest{
			Identifier: "nonexistent@example.com",
			Password:   "password123",
		}

		tokens, userID, err := service.Login(req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Zero(t, userID)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("user not found by username", func(t *testing.T) {
		req := &dto.LoginRequest{
			Identifier: "nonexistentuser",
			Password:   "password123",
		}

		tokens, userID, err := service.Login(req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Zero(t, userID)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("incorrect password with email", func(t *testing.T) {
		req := &dto.LoginRequest{
			Identifier: "login@example.com",
			Password:   "wrongpassword",
		}

		tokens, userID, err := service.Login(req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Zero(t, userID)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	t.Run("incorrect password with username", func(t *testing.T) {
		req := &dto.LoginRequest{
			Identifier: "testuser",
			Password:   "wrongpassword",
		}

		tokens, userID, err := service.Login(req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Zero(t, userID)
		assert.Contains(t, err.Error(), "invalid credentials")
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	service := getTestService(t)

	registerReq := &dto.RegisterRequest{
		Email:     "refresh@example.com",
		Password:  "password123",
		FirstName: "Refresh",
		LastName:  "Test",
		Username:  "refreshtest",
	}
	tokens, _, err := service.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful token refresh", func(t *testing.T) {
		time.Sleep(time.Second)

		newTokens, err := service.RefreshToken(tokens.RefreshToken)

		assert.NoError(t, err)
		assert.NotNil(t, newTokens)
		assert.NotEmpty(t, newTokens.AccessToken)
		assert.NotEmpty(t, newTokens.RefreshToken)
		assert.NotEqual(t, tokens.AccessToken, newTokens.AccessToken)
	})
	t.Run("invalid token", func(t *testing.T) {
		newTokens, err := service.RefreshToken("invalid-token")

		assert.Error(t, err)
		assert.Nil(t, newTokens)
		assert.Contains(t, err.Error(), "invalid refresh token")
	})

	t.Run("wrong token type", func(t *testing.T) {
		newTokens, err := service.RefreshToken(tokens.AccessToken)

		assert.Error(t, err)
		assert.Nil(t, newTokens)
		assert.Contains(t, err.Error(), "invalid token type")
	})

	t.Run("expired token", func(t *testing.T) {
		claims := &JWTClaims{
			UserID: 1,
			Email:  "test@example.com",
			Type:   "refresh",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		expiredToken, _ := token.SignedString([]byte(cfg.JWTSecret))

		newTokens, err := service.RefreshToken(expiredToken)

		assert.Error(t, err)
		assert.Nil(t, newTokens)
	})
}

func TestAuthService_UpdatePassword(t *testing.T) {
	service := getTestService(t)

	registerReq := &dto.RegisterRequest{
		Email:     "password@example.com",
		Password:  "oldpassword",
		FirstName: "Password",
		LastName:  "Test",
		Username:  "passwordtest",
	}
	_, userID, err := service.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful password update", func(t *testing.T) {
		req := &dto.UpdatePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		}

		err := service.UpdatePassword(userID, req)
		assert.NoError(t, err)

		var auth model.Auth
		err = testDB.Where("user_id = ?", userID).First(&auth).Error
		require.NoError(t, err)

		err = bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte("newpassword123"))
		assert.NoError(t, err)
	})

	t.Run("incorrect current password", func(t *testing.T) {
		req := &dto.UpdatePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword123",
		}

		err := service.UpdatePassword(userID, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "current password is incorrect")
	})

	t.Run("user not found", func(t *testing.T) {
		req := &dto.UpdatePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		}

		err := service.UpdatePassword(9999, req)

		assert.Error(t, err)
	})
}

func TestAuthService_UpdateEmail(t *testing.T) {
	service := getTestService(t)

	registerReq := &dto.RegisterRequest{
		Email:     "oldemail@example.com",
		Password:  "password123",
		FirstName: "Email",
		LastName:  "Test",
		Username:  "emailtest",
	}
	_, userID, err := service.Register(registerReq)
	require.NoError(t, err)

	t.Run("successful email update", func(t *testing.T) {
		req := &dto.UpdateEmailRequest{
			Password: "password123",
			NewEmail: "newemail@example.com",
		}

		tokens, err := service.UpdateEmail(userID, req)

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)

		var auth model.Auth
		err = testDB.Where("user_id = ?", userID).First(&auth).Error
		require.NoError(t, err)
		assert.Equal(t, "newemail@example.com", auth.Email)
	})

	t.Run("incorrect password", func(t *testing.T) {
		req := &dto.UpdateEmailRequest{
			Password: "wrongpassword",
			NewEmail: "another@example.com",
		}

		tokens, err := service.UpdateEmail(userID, req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Contains(t, err.Error(), "password is incorrect")
	})

	t.Run("same email", func(t *testing.T) {
		req := &dto.UpdateEmailRequest{
			Password: "password123",
			NewEmail: "newemail@example.com",
		}

		tokens, err := service.UpdateEmail(userID, req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Contains(t, err.Error(), "new email is the same as current email")
	})

	t.Run("email already in use", func(t *testing.T) {
		otherReq := &dto.RegisterRequest{
			Email:     "existing@example.com",
			Password:  "password123",
			FirstName: "Other",
			LastName:  "User",
			Username:  "existinguser",
		}
		_, _, err := service.Register(otherReq)
		require.NoError(t, err)

		req := &dto.UpdateEmailRequest{
			Password: "password123",
			NewEmail: "existing@example.com",
		}

		tokens, err := service.UpdateEmail(userID, req)

		assert.Error(t, err)
		assert.Nil(t, tokens)
		assert.Contains(t, err.Error(), "email already in use")
	})
}

func TestAuthService_TokenGeneration(t *testing.T) {
	service := getTestService(t)

	t.Run("generates valid tokens with correct claims", func(t *testing.T) {
		tokens, err := service.generateTokens(1, "test@example.com")

		assert.NoError(t, err)
		assert.NotNil(t, tokens)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, "Bearer", tokens.TokenType)
		assert.Greater(t, tokens.ExpiresIn, int64(0))

		accessClaims, err := service.validateToken(tokens.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), accessClaims.UserID)
		assert.Equal(t, "test@example.com", accessClaims.Email)
		assert.Equal(t, "access", accessClaims.Type)

		refreshClaims, err := service.validateToken(tokens.RefreshToken)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), refreshClaims.UserID)
		assert.Equal(t, "test@example.com", refreshClaims.Email)
		assert.Equal(t, "refresh", refreshClaims.Type)
	})
}

func TestAuthService_TokenValidation(t *testing.T) {
	service := getTestService(t)

	t.Run("validates correct token", func(t *testing.T) {
		claims := &JWTClaims{
			UserID: 1,
			Email:  "test@example.com",
			Type:   "access",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(cfg.JWTSecret))

		validatedClaims, err := service.validateToken(tokenString)

		assert.NoError(t, err)
		assert.NotNil(t, validatedClaims)
		assert.Equal(t, uint(1), validatedClaims.UserID)
		assert.Equal(t, "test@example.com", validatedClaims.Email)
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		claims, err := service.validateToken("not-a-valid-token")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("rejects token with wrong secret", func(t *testing.T) {
		claims := &JWTClaims{
			UserID: 1,
			Email:  "test@example.com",
			Type:   "access",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("wrong-secret"))

		validatedClaims, err := service.validateToken(tokenString)

		assert.Error(t, err)
		assert.Nil(t, validatedClaims)
	})
}
