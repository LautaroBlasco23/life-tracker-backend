package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"life-tracker-backend/internal/domain/user/dto"
	"life-tracker-backend/internal/domain/user/model"
	"life-tracker-backend/internal/infrastructure/imagestore"

	"gorm.io/gorm"
)

type UserService struct {
	db          *gorm.DB
	imageClient *imagestore.Client
}

func NewUserService(db *gorm.DB, imageClient *imagestore.Client) *UserService {
	return &UserService{
		db:          db,
		imageClient: imageClient,
	}
}

func (s *UserService) GetProfile(userID uint) (*dto.UserResponse, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, errors.New("failed to fetch user")
	}

	return user.ToResponse(), nil
}

func (s *UserService) UpdateProfile(userID uint, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	updates := make(map[string]interface{})
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.ProfilePicURL != nil {
		updates["profile_pic_url"] = *req.ProfilePicURL
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, errors.New("failed to update user")
		}
	}

	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("failed to fetch updated user")
	}

	return user.ToResponse(), nil
}

func (s *UserService) GetAllUsers() ([]*dto.UserResponse, error) {
	var users []model.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, errors.New("failed to fetch users")
	}

	responses := make([]*dto.UserResponse, 0, len(users))
	for i := range users {
		responses = append(responses, users[i].ToResponse())
	}

	return responses, nil
}

func (s *UserService) DeleteUser(userID uint) error {
	result := s.db.Delete(&model.User{}, userID)
	if result.Error != nil {
		return errors.New("failed to delete user")
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

// Image related methods
func (s *UserService) UploadProfileImage(ctx context.Context, userID uint, file *multipart.FileHeader) (*dto.UserResponse, error) {
	if err := s.validateImageFile(file); err != nil {
		return nil, err
	}

	src, err := file.Open()
	if err != nil {
		return nil, errors.New("failed to open uploaded file")
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	imageData, err := io.ReadAll(src)
	if err != nil {
		return nil, errors.New("failed to read file data")
	}

	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	originalURL, thumbnailURL, err := s.imageClient.UploadProfileImage(ctx, userID, imageData, file.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	if user.ProfilePicURL != nil && *user.ProfilePicURL != "" {
		oldImageID := s.extractImageIDFromURL(*user.ProfilePicURL)
		if oldImageID != "" {
			_ = s.imageClient.DeleteImage(ctx, oldImageID)
		}
	}

	user.ProfilePicURL = &originalURL
	user.ThumbnailURL = &thumbnailURL

	if err := s.db.Save(&user).Error; err != nil {
		return nil, errors.New("failed to update user profile")
	}

	return user.ToResponse(), nil
}

func (s *UserService) DeleteProfileImage(ctx context.Context, userID uint) error {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.New("user not found")
	}

	if user.ProfilePicURL == nil || *user.ProfilePicURL == "" {
		return errors.New("no profile image to delete")
	}

	imageID := s.extractImageIDFromURL(*user.ProfilePicURL)
	if imageID != "" {
		if err := s.imageClient.DeleteImage(ctx, imageID); err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}
	}

	user.ProfilePicURL = nil
	user.ThumbnailURL = nil

	if err := s.db.Save(&user).Error; err != nil {
		return errors.New("failed to update user profile")
	}

	return nil
}

func (s *UserService) validateImageFile(file *multipart.FileHeader) error {
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

func (s *UserService) extractImageIDFromURL(url string) string {
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
