package dto

import "time"

type CreateUserRequest struct {
	Timezone      *string `json:"timezone,omitempty"`
	FirstName     string  `json:"firstName" binding:"required,min=2,max=50"`
	LastName      string  `json:"lastName" binding:"required,min=2,max=50"`
	Email         string  `json:"email" binding:"required,email"`
	Password      string  `json:"password" binding:"required,min=6"`
	ProfilePicURL string  `json:"profilePicUrl,omitempty"`
}

type UpdateUserRequest struct {
	FirstName            *string `json:"firstName,omitempty" binding:"omitempty,min=2,max=50"`
	LastName             *string `json:"lastName,omitempty" binding:"omitempty,min=2,max=50"`
	ProfilePicURL        *string `json:"profilePicUrl,omitempty"`
	Timezone             *string `json:"timezone,omitempty" binding:"omitempty,timezone"`
	ProfilePrivacyStatus *string `json:"profilePrivacyStatus,omitempty" binding:"omitempty,oneof=public private"`
}

type UserResponse struct {
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	ProfilePicURL        *string   `json:"profilePicUrl,omitempty"`
	ThumbnailURL         *string   `json:"thumbnailUrl,omitempty"`
	Timezone             *string   `json:"timezone,omitempty"`
	Username             string    `json:"username"`
	FirstName            string    `json:"firstName"`
	LastName             string    `json:"lastName"`
	Email                string    `json:"email"`
	ProfilePrivacyStatus string    `json:"profilePrivacyStatus,omitempty"`
	ID                   uint      `json:"id"`
}

type PublicUserCard struct {
	ProfilePicURL        *string `json:"profilePicUrl,omitempty"`
	Username             string  `json:"username"`
	FirstName            string  `json:"firstName"`
	LastName             string  `json:"lastName"`
	ProfilePrivacyStatus string  `json:"profilePrivacyStatus"`
	FollowStatus         string  `json:"followStatus"`
	ID                   uint    `json:"id"`
	IsFollowing          bool    `json:"isFollowing"`
}

type PublicProfileResponse struct {
	PublicUserCard
	FollowersCount        int64 `json:"followersCount"`
	FollowingCount        int64 `json:"followingCount"`
	PublicRoutinesCount   int64 `json:"publicRoutinesCount"`
	PublicActivitiesCount int64 `json:"publicActivitiesCount"`
	CanViewContent        bool  `json:"canViewContent"`
}

type UserSearchQuery struct {
	Q     string `form:"q" binding:"required,min=1,max=30"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=50"`
}
