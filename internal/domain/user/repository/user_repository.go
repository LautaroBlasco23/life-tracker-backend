package repository

import (
	"errors"

	"life-tracker-backend/internal/domain/user/model"

	"gorm.io/gorm"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already taken")
)

type UserRepository interface {
	Create(user *model.User) error
	FindByID(id uint) (*model.User, error)
	Update(user *model.User, updates map[string]interface{}) error
	Delete(id uint) error
	FindAll() ([]model.User, error)
	FindByUsername(username string) (*model.User, error)
	SearchByUsernamePrefix(prefix string, limit int) ([]model.User, error)
	UsernameExists(username string) (bool, error)
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *GormUserRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) Update(user *model.User, updates map[string]interface{}) error {
	return r.db.Model(user).Updates(updates).Error
}

func (r *GormUserRepository) Delete(id uint) error {
	result := r.db.Delete(&model.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *GormUserRepository) FindAll() ([]model.User, error) {
	var users []model.User
	if err := r.db.Where("deleted_at IS NULL").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *GormUserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("LOWER(username) = LOWER(?) AND deleted_at IS NULL", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) SearchByUsernamePrefix(prefix string, limit int) ([]model.User, error) {
	if limit <= 0 {
		limit = 25
	}
	var users []model.User
	err := r.db.Where("username ILIKE ? AND deleted_at IS NULL", prefix+"%").Limit(limit).Find(&users).Error
	return users, err
}

func (r *GormUserRepository) UsernameExists(username string) (bool, error) {
	var user model.User
	err := r.db.Where("LOWER(username) = LOWER(?)", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
