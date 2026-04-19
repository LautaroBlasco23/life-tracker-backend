package dto

import "time"

type ActivityIDPosition struct {
	ActivityID uint `json:"activityId" binding:"required"`
	Position   int  `json:"position"`
}

type CreateRoutineRequest struct {
	Title         string               `json:"title" binding:"required,min=1,max=255"`
	Description   string               `json:"description" binding:"max=1000"`
	PrivacyStatus string               `json:"privacyStatus" binding:"required,oneof=public private"`
	ActivityIDs   []ActivityIDPosition `json:"activityIds" binding:"required,min=1"`
}

type UpdateRoutineRequest struct {
	Title         *string              `json:"title,omitempty" binding:"omitempty,min=1,max=255"`
	Description   *string              `json:"description,omitempty" binding:"omitempty,max=1000"`
	PrivacyStatus *string              `json:"privacyStatus,omitempty" binding:"omitempty,oneof=public private"`
	ActivityIDs   []ActivityIDPosition `json:"activityIds,omitempty"`
}

type RoutineActivityItem struct {
	ActivityTitle string `json:"activityTitle"`
	ActivityID    uint   `json:"activityId"`
	Position      int    `json:"position"`
}

type RoutineResponse struct {
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`
	Title         string                `json:"title"`
	Description   string                `json:"description"`
	PrivacyStatus string                `json:"privacyStatus"`
	Activities    []RoutineActivityItem `json:"activities"`
	ID            uint                  `json:"id"`
	UserID        uint                  `json:"userId"`
}
