package dto

import "time"

type CreateActivityRequest struct {
	Title            string `json:"title" binding:"required,min=1,max=255"`
	Description      string `json:"description" binding:"max=1000"`
	CompletionAmount int    `json:"completionAmount" binding:"required,min=1"`
	Frequency        string `json:"frequency" binding:"required,oneof=daily weekly monthly oneTime"`
	DayFrequency     string `json:"dayFrequency,omitempty"` // JSON array: ["monday", "wednesday", "friday"]
	DayTime          string `json:"dayTime" binding:"required,oneof=morning afternoon evening"`
}

type UpdateActivityRequest struct {
	Title            *string `json:"title,omitempty" binding:"omitempty,min=1,max=255"`
	Description      *string `json:"description,omitempty" binding:"omitempty,max=1000"`
	CompletionAmount *int    `json:"completionAmount,omitempty" binding:"omitempty,min=1"`
	Frequency        *string `json:"frequency,omitempty" binding:"omitempty,oneof=daily weekly monthly oneTime"`
	DayFrequency     *string `json:"dayFrequency,omitempty"`
	DayTime          *string `json:"dayTime,omitempty" binding:"omitempty,oneof=morning afternoon evening"`
	IsActive         *bool   `json:"isActive,omitempty"`
}

type ActivityResponse struct {
	ID               uint      `json:"id"`
	UserID           uint      `json:"userId"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	CompletionAmount int       `json:"completionAmount"`
	Frequency        string    `json:"frequency"`
	DayFrequency     string    `json:"dayFrequency,omitempty"`
	DayTime          string    `json:"dayTime"`
	IsActive         bool      `json:"isActive"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	TodayCompletions int       `json:"todayCompletions"`
	IsCompletedToday bool      `json:"isCompletedToday"`
}

// Activity completion DTOs
type RecordActivityRequest struct {
	CompletionDate time.Time `json:"completionDate,omitempty"` // Optional, defaults to now
	Notes          string    `json:"notes,omitempty" binding:"max=500"`
}

type ActivityRecordResponse struct {
	ID             string    `json:"id"`
	ActivityID     uint      `json:"activityId"`
	UserID         uint      `json:"userId"`
	CompletionDate time.Time `json:"completionDate"`
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ActivityStatsResponse struct {
	ActivityID       uint                      `json:"activityId"`
	Title            string                    `json:"title"`
	TotalCompletions int64                     `json:"totalCompletions"`
	CurrentStreak    int                       `json:"currentStreak"`
	LongestStreak    int                       `json:"longestStreak"`
	CompletionRate   float64                   `json:"completionRate"` // Percentage
	RecentRecords    []*ActivityRecordResponse `json:"recentRecords"`
}
