package repository

import (
	"errors"

	"life-tracker-backend/internal/domain/auth/model"
	userModel "life-tracker-backend/internal/domain/user/model"

	"gorm.io/gorm"
)

var (
	ErrAuthNotFound  = errors.New("auth not found")
	ErrUserNotFound  = errors.New("user not found")
	ErrUserExists    = errors.New("user already exists")
	ErrUsernameTaken = errors.New("username already taken")
)

type AuthRepository interface {
	FindByEmail(email string) (*model.Auth, error)
	FindByUsername(username string) (*model.Auth, error)
	FindByUserID(userID uint) (*model.Auth, error)
	Create(auth *model.Auth) error
	EmailExists(email string) (bool, error)
	UsernameExists(username string) (bool, error)
	SearchByUsernamePrefix(prefix string, limit int) ([]model.Auth, error)
	UpdatePassword(userID uint, passwordHash string) error
	UpdateEmail(userID uint, email string) error
	UpdateUsername(userID uint, username string) error
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

func (r *GormAuthRepository) FindByUsername(username string) (*model.Auth, error) {
	var auth model.Auth
	err := r.db.Where("LOWER(username) = LOWER(?)", username).First(&auth).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAuthNotFound
		}
		return nil, err
	}
	return &auth, nil
}

func (r *GormAuthRepository) UsernameExists(username string) (bool, error) {
	var auth model.Auth
	err := r.db.Where("LOWER(username) = LOWER(?)", username).First(&auth).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *GormAuthRepository) SearchByUsernamePrefix(prefix string, limit int) ([]model.Auth, error) {
	if limit <= 0 {
		limit = 25
	}
	var auths []model.Auth
	err := r.db.Where("username ILIKE ?", prefix+"%").Limit(limit).Find(&auths).Error
	return auths, err
}

func (r *GormAuthRepository) FindByUserID(userID uint) (*model.Auth, error) {
	var auth model.Auth
	err := r.db.Where("user_id = ?", userID).First(&auth).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAuthNotFound
		}
		return nil, err
	}
	return &auth, nil
}

func (r *GormAuthRepository) UpdatePassword(userID uint, passwordHash string) error {
	result := r.db.Model(&model.Auth{}).Where("user_id = ?", userID).Update("password_hash", passwordHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAuthNotFound
	}
	return nil
}

func (r *GormAuthRepository) UpdateEmail(userID uint, email string) error {
	result := r.db.Model(&model.Auth{}).Where("user_id = ?", userID).Update("email", email)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAuthNotFound
	}
	return nil
}

func (r *GormAuthRepository) UpdateUsername(userID uint, username string) error {
	result := r.db.Model(&model.Auth{}).Where("user_id = ?", userID).Update("username", username)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAuthNotFound
	}
	return nil
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(user *userModel.User) error {
	if user.FirstName == "" {
		return errors.New("first name is required")
	}
	if user.LastName == "" {
		return errors.New("last name is required")
	}
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
