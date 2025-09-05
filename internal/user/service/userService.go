package service

import (
	"errors"
	"life-tracker-backend/internal/user/model"

	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		db: db,
	}
}

func (s *UserService) GetProfile(userID uint) (*model.UserResponse, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	return user.ToResponse(), nil
}

func (s *UserService) UpdateProfile(userID uint, req *model.UpdateUserRequest) (*model.UserResponse, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// Update only provided fields
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

	// Fetch updated user
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, errors.New("failed to fetch updated user")
	}

	return user.ToResponse(), nil
}

func (s *UserService) GetAllUsers() ([]*model.UserResponse, error) {
	var users []model.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, errors.New("failed to fetch users")
	}

	var responses []*model.UserResponse
	for _, user := range users {
		responses = append(responses, user.ToResponse())
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
