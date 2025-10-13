package service

import (
	"errors"

	"life-tracker-backend/internal/domain/user/dto"
	"life-tracker-backend/internal/domain/user/model"

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
