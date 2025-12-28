package dto

import "time"

type CreateActivityRequest struct {
	Title            string `json:"title" binding:"required,min=1,max=255"`
	Description      string `json:"description" binding:"max=1000"`
	Frequency        string `json:"frequency" binding:"required,oneof=daily weekly monthly oneTime"`
	DayFrequency     string `json:"dayFrequency,omitempty"`
	DayTime          string `json:"dayTime" binding:"required,oneof=morning afternoon evening"`
	CompletionAmount int    `json:"completionAmount" binding:"required,min=1"`
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
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
	Description      string      `json:"description"`
	Frequency        string      `json:"frequency"`
	DayFrequency     string      `json:"dayFrequency,omitempty"`
	DayTime          string      `json:"dayTime"`
	Title            string      `json:"title"`
	ID               uint        `json:"id"`
	CompletionAmount int         `json:"completionAmount"`
	UserID           uint        `json:"userId"`
	TodayCompletions int         `json:"todayCompletions"`
	IsActive         bool        `json:"isActive"`
	IsCompletedToday bool        `json:"isCompletedToday"`
	Streak           *StreakInfo `json:"streak,omitempty"`
}

// Activity completion DTOs
type RecordActivityRequest struct {
	CompletionDate time.Time `json:"completionDate,omitempty"` // Optional, defaults to now
	Notes          string    `json:"notes,omitempty" binding:"max=500"`
}

type ActivityRecordResponse struct {
	CompletionDate time.Time `json:"completionDate"`
	CreatedAt      time.Time `json:"createdAt"`
	ID             string    `json:"id"`
	Notes          string    `json:"notes,omitempty"`
	ActivityID     uint      `json:"activityId"`
	UserID         uint      `json:"userId"`
}

type ActivityStatsResponse struct {
	Title            string                    `json:"title"`
	RecentRecords    []*ActivityRecordResponse `json:"recentRecords"`
	ActivityID       uint                      `json:"activityId"`
	TotalCompletions int64                     `json:"totalCompletions"`
	CurrentStreak    int                       `json:"currentStreak"`
	LongestStreak    int                       `json:"longestStreak"`
	CompletionRate   float64                   `json:"completionRate"`
}

type ActivityFilter struct {
	Frequency    string `form:"frequency"`
	DayTime      string `form:"day_time"`
	ScheduledFor string `form:"scheduled_for"`
}

type RevertCompletionRequest struct {
	TargetDate *time.Time `json:"targetDate,omitempty"`
}

type StreakInfo struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
}
