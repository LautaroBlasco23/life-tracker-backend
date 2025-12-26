package dto

import "time"

type CreateTimeRecordRequest struct {
	Category        string `json:"category" binding:"required,min=1,max=100"`
	Description     string `json:"description" binding:"required,max=1000"`
	DurationMinutes int    `json:"durationMinutes" binding:"required,min=1"`
}

type UpdateTimeRecordRequest struct {
	Category        *string `json:"category,omitempty" binding:"omitempty,min=1,max=100"`
	Description     *string `json:"description,omitempty" binding:"omitempty,max=1000"`
	DurationMinutes *int    `json:"durationMinutes,omitempty" binding:"omitempty,min=1"`
}

type TimeRecordResponse struct {
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	Category        string    `json:"category"`
	Description     string    `json:"description"`
	ID              uint      `json:"id"`
	DurationMinutes int       `json:"durationMinutes"`
}

type TimeRecordFilter struct {
	StartDate *time.Time `form:"-"`
	EndDate   *time.Time `form:"-"`
	Month     *int       `form:"-"`
	Year      *int       `form:"-"`
	Category  string     `form:"category"`
}

type TimeStatsResponse struct {
	CategoryTotals  map[string]int `json:"categoryTotals"`
	TopCategory     string         `json:"topCategory,omitempty"`
	TotalMinutes    int            `json:"totalMinutes"`
	RecordCount     int            `json:"recordCount"`
	TopCategoryMins int            `json:"topCategoryMinutes,omitempty"`
}
