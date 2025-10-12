package model

import (
	"life-tracker-backend/internal/domain/user/dto"
	"time"

	"gorm.io/gorm"
)

type User struct {
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	ProfilePicURL *string        `json:"profilePicUrl,omitempty"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
	FirstName     string         `json:"firstName" gorm:"not null"`
	LastName      string         `json:"lastName" gorm:"not null"`
	Email         string         `json:"email" gorm:"uniqueIndex;not null"`
	ID            uint           `json:"id" gorm:"primaryKey"`
}

func (u *User) ToResponse() *dto.UserResponse {
	return &dto.UserResponse{
		ID:            u.ID,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Email:         u.Email,
		ProfilePicURL: u.ProfilePicURL,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
