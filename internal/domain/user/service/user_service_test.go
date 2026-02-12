package service

import (
	"fmt"
	"mime/multipart"
	"os"
	"testing"
	"time"

	"life-tracker-backend/internal/config"
	"life-tracker-backend/internal/domain/user/dto"
	"life-tracker-backend/internal/domain/user/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		require.NoError(t, db.AutoMigrate(&model.User{}))
		testDB = db
	}
}

func cleanDatabase(t *testing.T) {
	t.Helper()
	require.NoError(t, testDB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE;").Error)
}

func getTestService(t *testing.T) *UserService {
	setupTestDatabase(t)
	cleanDatabase(t)
	return NewUserService(testDB, nil)
}

func createTestUser(t *testing.T, service *UserService) uint {
	user := &model.User{
		FirstName: "Test",
		LastName:  "User",
	}
	err := testDB.Create(user).Error
	require.NoError(t, err)
	return user.ID
}

func TestUserService_GetMyProfile(t *testing.T) {
	service := getTestService(t)

	t.Run("get existing user profile", func(t *testing.T) {
		userID := createTestUser(t, service)
		email := "test@example.com"

		profile, err := service.GetMyProfile(userID, email)

		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, userID, profile.ID)
		assert.Equal(t, "Test", profile.FirstName)
		assert.Equal(t, "User", profile.LastName)
		assert.Equal(t, email, profile.Email)
	})

	t.Run("user not found", func(t *testing.T) {
		profile, err := service.GetMyProfile(9999, "test@example.com")

		assert.Error(t, err)
		assert.Nil(t, profile)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestUserService_GetUserByID(t *testing.T) {
	service := getTestService(t)

	t.Run("get existing user by id", func(t *testing.T) {
		userID := createTestUser(t, service)

		user, err := service.GetUserByID(userID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "Test", user.FirstName)
		assert.Empty(t, user.Email)
	})

	t.Run("user not found", func(t *testing.T) {
		user, err := service.GetUserByID(9999)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestUserService_GetUserTimezone(t *testing.T) {
	service := getTestService(t)

	t.Run("user with timezone", func(t *testing.T) {
		tz := "America/New_York"
		user := &model.User{
			FirstName: "Test",
			LastName:  "User",
			Timezone:  &tz,
		}
		err := testDB.Create(user).Error
		require.NoError(t, err)

		loc, err := service.GetUserTimezone(user.ID)

		assert.NoError(t, err)
		assert.Equal(t, "America/New_York", loc.String())
	})

	t.Run("user without timezone defaults to UTC", func(t *testing.T) {
		userID := createTestUser(t, service)

		loc, err := service.GetUserTimezone(userID)

		assert.NoError(t, err)
		assert.Equal(t, "UTC", loc.String())
	})

	t.Run("user not found defaults to UTC", func(t *testing.T) {
		loc, err := service.GetUserTimezone(9999)

		assert.NoError(t, err)
		assert.Equal(t, "UTC", loc.String())
	})

	t.Run("invalid timezone defaults to UTC", func(t *testing.T) {
		invalidTZ := "Invalid/Timezone"
		user := &model.User{
			FirstName: "Test",
			LastName:  "User",
			Timezone:  &invalidTZ,
		}
		err := testDB.Create(user).Error
		require.NoError(t, err)

		loc, err := service.GetUserTimezone(user.ID)

		assert.NoError(t, err)
		assert.Equal(t, "UTC", loc.String())
	})
}

func TestUserService_UpdateProfile(t *testing.T) {
	service := getTestService(t)

	t.Run("update first name and last name", func(t *testing.T) {
		userID := createTestUser(t, service)
		email := "test@example.com"

		newFirstName := "Updated"
		newLastName := "Name"
		req := &dto.UpdateUserRequest{
			FirstName: &newFirstName,
			LastName:  &newLastName,
		}

		profile, err := service.UpdateProfile(userID, email, req)

		assert.NoError(t, err)
		assert.Equal(t, "Updated", profile.FirstName)
		assert.Equal(t, "Name", profile.LastName)

		var user model.User
		err = testDB.First(&user, userID).Error
		require.NoError(t, err)
		assert.Equal(t, "Updated", user.FirstName)
		assert.Equal(t, "Name", user.LastName)
	})

	t.Run("update timezone", func(t *testing.T) {
		userID := createTestUser(t, service)
		email := "test@example.com"

		timezone := "Europe/London"
		req := &dto.UpdateUserRequest{
			Timezone: &timezone,
		}

		profile, err := service.UpdateProfile(userID, email, req)

		assert.NoError(t, err)
		assert.NotNil(t, profile.Timezone)
		assert.Equal(t, "Europe/London", *profile.Timezone)
	})

	t.Run("invalid timezone", func(t *testing.T) {
		userID := createTestUser(t, service)
		email := "test@example.com"

		invalidTZ := "Invalid/Zone"
		req := &dto.UpdateUserRequest{
			Timezone: &invalidTZ,
		}

		profile, err := service.UpdateProfile(userID, email, req)

		assert.Error(t, err)
		assert.Nil(t, profile)
		assert.Contains(t, err.Error(), "invalid timezone")
	})

	t.Run("empty update returns unchanged profile", func(t *testing.T) {
		userID := createTestUser(t, service)
		email := "test@example.com"

		req := &dto.UpdateUserRequest{}

		profile, err := service.UpdateProfile(userID, email, req)

		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, "Test", profile.FirstName)
	})

	t.Run("user not found", func(t *testing.T) {
		newFirstName := "Updated"
		req := &dto.UpdateUserRequest{
			FirstName: &newFirstName,
		}

		profile, err := service.UpdateProfile(9999, "test@example.com", req)

		assert.Error(t, err)
		assert.Nil(t, profile)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestUserService_GetAllUsers(t *testing.T) {
	service := getTestService(t)

	t.Run("get all users", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			user := &model.User{
				FirstName: fmt.Sprintf("User%d", i),
				LastName:  "Test",
			}
			err := testDB.Create(user).Error
			require.NoError(t, err)
		}

		users, err := service.GetAllUsers()

		assert.NoError(t, err)
		assert.Len(t, users, 3)
	})

	t.Run("no users returns empty list", func(t *testing.T) {
		users, err := service.GetAllUsers()

		assert.NoError(t, err)
		assert.Empty(t, users)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	service := getTestService(t)

	t.Run("delete existing user", func(t *testing.T) {
		userID := createTestUser(t, service)

		err := service.DeleteUser(userID)

		assert.NoError(t, err)

		var user model.User
		err = testDB.Unscoped().First(&user, userID).Error
		assert.NoError(t, err)
		assert.NotNil(t, user.DeletedAt)
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		err := service.DeleteUser(9999)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})
}

func TestUserService_ValidateImageFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		size     int64
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid jpg file",
			filename: "test.jpg",
			size:     1024,
			wantErr:  false,
		},
		{
			name:     "valid png file",
			filename: "test.png",
			size:     1024,
			wantErr:  false,
		},
		{
			name:     "valid webp file",
			filename: "test.webp",
			size:     1024,
			wantErr:  false,
		},
		{
			name:     "file too large",
			filename: "test.jpg",
			size:     11 * 1024 * 1024,
			wantErr:  true,
			errMsg:   "file size exceeds 10MB limit",
		},
		{
			name:     "invalid file type",
			filename: "test.pdf",
			size:     1024,
			wantErr:  true,
			errMsg:   "invalid file type",
		},
		{
			name:     "invalid file type exe",
			filename: "test.exe",
			size:     1024,
			wantErr:  true,
			errMsg:   "invalid file type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := &multipart.FileHeader{
				Filename: tt.filename,
				Size:     tt.size,
			}

			err := validateImageFile(file)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserService_ExtractImageIDFromURL(t *testing.T) {
	tests := []struct {
		url      string
		name     string
		expected string
	}{
		{
			name:     "standard URL",
			url:      "https://storage.example.com/images/user123.jpg",
			expected: "user123.jpg",
		},
		{
			name:     "URL with query params",
			url:      "https://storage.example.com/images/user123.jpg?token=abc123",
			expected: "user123.jpg",
		},
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "URL with multiple paths",
			url:      "https://cdn.example.com/v1/users/profiles/image123.png",
			expected: "image123.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractImageIDFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserService_BuildUserUpdates(t *testing.T) {
	t.Run("all fields provided", func(t *testing.T) {
		firstName := "John"
		lastName := "Doe"
		profilePic := "http://example.com/pic.jpg"
		timezone := "America/New_York"

		req := &dto.UpdateUserRequest{
			FirstName:     &firstName,
			LastName:      &lastName,
			ProfilePicURL: &profilePic,
			Timezone:      &timezone,
		}

		updates := buildUserUpdates(req)

		assert.Len(t, updates, 4)
		assert.Equal(t, "John", updates["first_name"])
		assert.Equal(t, "Doe", updates["last_name"])
		assert.Equal(t, "http://example.com/pic.jpg", updates["profile_pic_url"])
		assert.Equal(t, "America/New_York", updates["timezone"])
	})

	t.Run("partial fields", func(t *testing.T) {
		firstName := "John"
		req := &dto.UpdateUserRequest{
			FirstName: &firstName,
		}

		updates := buildUserUpdates(req)

		assert.Len(t, updates, 1)
		assert.Equal(t, "John", updates["first_name"])
	})

	t.Run("no fields", func(t *testing.T) {
		req := &dto.UpdateUserRequest{}

		updates := buildUserUpdates(req)

		assert.Empty(t, updates)
	})
}

func TestUserModel_ToResponse(t *testing.T) {
	t.Run("convert user to response with email", func(t *testing.T) {
		profilePic := "http://example.com/pic.jpg"
		timezone := "America/New_York"

		user := &model.User{
			ID:            1,
			FirstName:     "John",
			LastName:      "Doe",
			ProfilePicURL: &profilePic,
			Timezone:      &timezone,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		response := user.ToResponse("john@example.com")

		assert.Equal(t, uint(1), response.ID)
		assert.Equal(t, "John", response.FirstName)
		assert.Equal(t, "Doe", response.LastName)
		assert.Equal(t, "john@example.com", response.Email)
		assert.Equal(t, &profilePic, response.ProfilePicURL)
		assert.Equal(t, &timezone, response.Timezone)
	})

	t.Run("convert user to response without email", func(t *testing.T) {
		user := &model.User{
			ID:        1,
			FirstName: "John",
			LastName:  "Doe",
		}

		response := user.ToResponse("")

		assert.Equal(t, "", response.Email)
	})
}

func TestUserModel_GetTimezoneLocation(t *testing.T) {
	t.Run("valid timezone", func(t *testing.T) {
		tz := "America/New_York"
		user := &model.User{
			Timezone: &tz,
		}

		loc := user.GetTimezoneLocation()

		assert.Equal(t, "America/New_York", loc.String())
	})

	t.Run("nil timezone defaults to UTC", func(t *testing.T) {
		user := &model.User{}

		loc := user.GetTimezoneLocation()

		assert.Equal(t, "UTC", loc.String())
	})

	t.Run("empty timezone defaults to UTC", func(t *testing.T) {
		emptyTZ := ""
		user := &model.User{
			Timezone: &emptyTZ,
		}

		loc := user.GetTimezoneLocation()

		assert.Equal(t, "UTC", loc.String())
	})

	t.Run("invalid timezone defaults to UTC", func(t *testing.T) {
		invalidTZ := "Invalid/Timezone"
		user := &model.User{
			Timezone: &invalidTZ,
		}

		loc := user.GetTimezoneLocation()

		assert.Equal(t, "UTC", loc.String())
	})
}
