package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"life-tracker-backend/internal/domain/user/dto"
	"life-tracker-backend/internal/domain/user/model"
	"life-tracker-backend/internal/domain/user/repository"
	"life-tracker-backend/internal/infrastructure/imagestore"
	"life-tracker-backend/internal/infrastructure/monitoring"

	"gorm.io/gorm"
)

type UserService struct {
	repo        repository.UserRepository
	imageClient *imagestore.Client
}

func NewUserService(db *gorm.DB, imageClient *imagestore.Client) *UserService {
	return &UserService{
		repo:        repository.NewUserRepository(db),
		imageClient: imageClient,
	}
}

func (s *UserService) GetMyProfile(userID uint, email string) (*dto.UserResponse, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("failed to fetch user")
	}
	return user.ToResponse(email), nil
}

func (s *UserService) GetUserByID(userID uint) (*dto.UserResponse, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("failed to fetch user")
	}
	return user.ToResponse(""), nil
}

func (s *UserService) GetUserTimezone(userID uint) (*time.Location, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return time.UTC, nil
		}
		return time.UTC, err
	}
	return user.GetTimezoneLocation(), nil
}

func (s *UserService) UpdateProfile(userID uint, email string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("failed to fetch user")
	}

	// Validate username format and uniqueness if being updated
	if req.Username != nil && *req.Username != user.Username {
		// Validate username format (lowercase letters, digits, underscores only)
		matched, regexErr := regexp.MatchString(`^[a-z0-9_]{3,30}$`, strings.ToLower(*req.Username))
		if regexErr != nil {
			return nil, errors.New("failed to validate username")
		}
		if !matched {
			return nil, errors.New("username must be 3-30 characters: lowercase letters, digits, underscores only")
		}

		exists, existsErr := s.repo.UsernameExists(*req.Username)
		if existsErr != nil {
			return nil, errors.New("failed to check username availability")
		}
		if exists {
			return nil, repository.ErrUsernameTaken
		}
	}

	updates := buildUserUpdates(req)
	if len(updates) == 0 {
		return user.ToResponse(email), nil
	}

	if req.Timezone != nil {
		if _, err = time.LoadLocation(*req.Timezone); err != nil {
			return nil, errors.New("invalid timezone")
		}
	}

	if err = s.repo.Update(user, updates); err != nil {
		return nil, errors.New("failed to update user")
	}

	user, err = s.repo.FindByID(userID)
	if err != nil {
		return nil, errors.New("failed to fetch updated user")
	}

	return user.ToResponse(email), nil
}

func buildUserUpdates(req *dto.UpdateUserRequest) map[string]interface{} {
	updates := make(map[string]interface{})
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.Username != nil {
		updates["username"] = strings.ToLower(*req.Username)
	}
	if req.ProfilePicURL != nil {
		updates["profile_pic_url"] = *req.ProfilePicURL
	}
	if req.Timezone != nil {
		updates["timezone"] = *req.Timezone
	}
	if req.ProfilePrivacyStatus != nil {
		updates["profile_privacy_status"] = *req.ProfilePrivacyStatus
	}
	return updates
}

func (s *UserService) GetAllUsers() ([]*dto.UserResponse, error) {
	users, err := s.repo.FindAll()
	if err != nil {
		return nil, errors.New("failed to fetch users")
	}

	responses := make([]*dto.UserResponse, 0, len(users))
	for i := range users {
		responses = append(responses, users[i].ToResponse(""))
	}
	return responses, nil
}

func (s *UserService) DeleteUser(userID uint) error {
	if err := s.repo.Delete(userID); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return errors.New("user not found")
		}
		return errors.New("failed to delete user")
	}
	monitoring.ActiveUsers.Dec()
	return nil
}

func (s *UserService) UploadProfileImage(ctx context.Context, userID uint, email string, file *multipart.FileHeader) (*dto.UserResponse, error) {
	if err := validateImageFile(file); err != nil {
		return nil, err
	}

	src, err := file.Open()
	if err != nil {
		return nil, errors.New("failed to open uploaded file")
	}
	defer func() {
		err = src.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	imageData, err := io.ReadAll(src)
	if err != nil {
		return nil, errors.New("failed to read file data")
	}

	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	var oldImageID string
	if user.ProfilePicURL != nil && *user.ProfilePicURL != "" {
		oldImageID = extractImageIDFromURL(*user.ProfilePicURL)
	}

	originalURL, thumbnailURL, err := s.imageClient.UploadProfileImage(ctx, userID, imageData, file.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	updates := map[string]interface{}{
		"profile_pic_url": originalURL,
		"thumbnail_url":   thumbnailURL,
	}
	if err = s.repo.Update(user, updates); err != nil {
		return nil, errors.New("failed to update user profile")
	}

	if oldImageID != "" {
		if err = s.imageClient.DeleteImage(ctx, oldImageID, fmt.Sprintf("%d", userID)); err != nil {
			fmt.Printf("warning: failed to delete old profile image %s: %v\n", oldImageID, err)
		}
	}

	monitoring.ProfileImageUploads.Inc()

	user, err = s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return user.ToResponse(email), nil
}

func (s *UserService) DeleteProfileImage(ctx context.Context, userID uint) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.ProfilePicURL == nil || *user.ProfilePicURL == "" {
		return errors.New("no profile image to delete")
	}

	imageID := extractImageIDFromURL(*user.ProfilePicURL)
	if imageID != "" {
		if err := s.imageClient.DeleteImage(ctx, imageID, fmt.Sprintf("%d", userID)); err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}
	}

	updates := map[string]interface{}{
		"profile_pic_url": nil,
		"thumbnail_url":   nil,
	}
	if err := s.repo.Update(user, updates); err != nil {
		return errors.New("failed to update user profile")
	}

	monitoring.ProfileImageDeletes.Inc()

	return nil
}

func validateImageFile(file *multipart.FileHeader) error {
	const maxSize = 10 * 1024 * 1024
	if file.Size > maxSize {
		return errors.New("file size exceeds 10MB limit")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}

	if !allowedExts[ext] {
		return errors.New("invalid file type, only JPG, PNG, and WebP are allowed")
	}

	return nil
}

func (s *UserService) SearchUsers(prefix string, limit int) ([]model.User, error) {
	return s.repo.SearchByUsernamePrefix(prefix, limit)
}

func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("failed to fetch user")
	}
	return user, nil
}

func (s *UserService) UsernameExists(username string) (bool, error) {
	return s.repo.UsernameExists(username)
}

func (s *UserService) EmailExists(email string) (bool, error) {
	return s.repo.EmailExists(email)
}

func extractImageIDFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	imageID := parts[len(parts)-1]
	if idx := strings.Index(imageID, "?"); idx != -1 {
		imageID = imageID[:idx]
	}
	return imageID
}
