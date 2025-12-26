package dto

import "time"

type CreateUserRequest struct {
	FirstName     string  `json:"firstName" binding:"required,min=2,max=50"`
	LastName      string  `json:"lastName" binding:"required,min=2,max=50"`
	Email         string  `json:"email" binding:"required,email"`
	Password      string  `json:"password" binding:"required,min=6"`
	ProfilePicURL string  `json:"profilePicUrl,omitempty"`
	Timezone      *string `json:"timezone,omitempty"`
}

type UpdateUserRequest struct {
	FirstName     *string `json:"firstName,omitempty" binding:"omitempty,min=2,max=50"`
	LastName      *string `json:"lastName,omitempty" binding:"omitempty,min=2,max=50"`
	ProfilePicURL *string `json:"profilePicUrl,omitempty"`
	Timezone      *string `json:"timezone,omitempty" binding:"omitempty,timezone"`
}

type UserResponse struct {
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ProfilePicURL *string   `json:"profilePicUrl,omitempty"`
	ThumbnailURL  *string   `json:"thumbnailUrl,omitempty"`
	Timezone      *string   `json:"timezone,omitempty"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Email         string    `json:"email"`
	ID            uint      `json:"id"`
}
