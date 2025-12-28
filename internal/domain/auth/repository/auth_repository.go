package repository

import (
	"errors"

	"life-tracker-backend/internal/domain/auth/model"
	userModel "life-tracker-backend/internal/domain/user/model"

	"gorm.io/gorm"
)

var (
	ErrAuthNotFound = errors.New("auth not found")
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type AuthRepository interface {
	FindByEmail(email string) (*model.Auth, error)
	Create(auth *model.Auth) error
	EmailExists(email string) (bool, error)
}

type UserRepository interface {
	Create(user *userModel.User) error
	FindByID(id uint) (*userModel.User, error)
}

type GormAuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &GormAuthRepository{db: db}
}

func (r *GormAuthRepository) FindByEmail(email string) (*model.Auth, error) {
	var auth model.Auth
	err := r.db.Where("email = ?", email).First(&auth).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAuthNotFound
		}
		return nil, err
	}
	return &auth, nil
}

func (r *GormAuthRepository) Create(auth *model.Auth) error {
	return r.db.Create(auth).Error
}

func (r *GormAuthRepository) EmailExists(email string) (bool, error) {
	var auth model.Auth
	err := r.db.Where("email = ?", email).First(&auth).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(user *userModel.User) error {
	return r.db.Create(user).Error
}

func (r *GormUserRepository) FindByID(id uint) (*userModel.User, error) {
	var user userModel.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
