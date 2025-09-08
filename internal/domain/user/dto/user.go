package dto

import "time"

type CreateUserRequest struct {
	FirstName     string `json:"firstName" binding:"required,min=2,max=50"`
	LastName      string `json:"lastName" binding:"required,min=2,max=50"`
	Email         string `json:"email" binding:"required,email"`
	Password      string `json:"password" binding:"required,min=6"`
	ProfilePicURL string `json:"profilePicUrl,omitempty"`
}

type UpdateUserRequest struct {
	FirstName     *string `json:"firstName,omitempty" binding:"omitempty,min=2,max=50"`
	LastName      *string `json:"lastName,omitempty" binding:"omitempty,min=2,max=50"`
	ProfilePicURL *string `json:"profilePicUrl,omitempty"`
}

type UserResponse struct {
	ID            uint      `json:"id"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Email         string    `json:"email"`
	ProfilePicURL *string   `json:"profilePicUrl,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
