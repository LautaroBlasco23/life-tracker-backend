package model

import (
	"time"

	"life-tracker-backend/internal/domain/user/dto"

	"gorm.io/gorm"
)

type User struct {
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
	ProfilePicURL        *string        `json:"profilePicUrl,omitempty"`
	ThumbnailURL         *string        `json:"thumbnailUrl,omitempty"`
	Timezone             *string        `json:"timezone,omitempty" gorm:"size:64"`
	ProfilePrivacyStatus string         `json:"profilePrivacyStatus" gorm:"not null;default:'public';size:20"`
	DeletedAt            gorm.DeletedAt `json:"-" gorm:"index"`
	FirstName            string         `json:"firstName" gorm:"not null"`
	LastName             string         `json:"lastName" gorm:"not null"`
	ID                   uint           `json:"id" gorm:"primaryKey"`
}

func (u *User) ToResponse(email, username string) *dto.UserResponse {
	return &dto.UserResponse{
		ID:                   u.ID,
		FirstName:            u.FirstName,
		LastName:             u.LastName,
		Email:                email,
		ProfilePicURL:        u.ProfilePicURL,
		ThumbnailURL:         u.ThumbnailURL,
		Timezone:             u.Timezone,
		Username:             username,
		ProfilePrivacyStatus: u.ProfilePrivacyStatus,
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
	}
}

func (u *User) GetTimezoneLocation() *time.Location {
	if u.Timezone == nil || *u.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(*u.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}
