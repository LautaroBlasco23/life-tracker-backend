package model

import (
	"database/sql/driver"
	"life-tracker-backend/internal/domain/activity/dto"
	"time"

	"gorm.io/gorm"
)

type Frequency string

const (
	FrequencyDaily   Frequency = "daily"
	FrequencyWeekly  Frequency = "weekly"
	FrequencyMonthly Frequency = "monthly"
	FrequencyOneTime Frequency = "oneTime"
)

func (f *Frequency) Scan(value interface{}) error {
	*f = Frequency(value.(string))
	return nil
}

func (f Frequency) Value() (driver.Value, error) {
	return string(f), nil
}

type DayOfWeek string

const (
	Monday    DayOfWeek = "monday"
	Tuesday   DayOfWeek = "tuesday"
	Wednesday DayOfWeek = "wednesday"
	Thursday  DayOfWeek = "thursday"
	Friday    DayOfWeek = "friday"
	Saturday  DayOfWeek = "saturday"
	Sunday    DayOfWeek = "sunday"
)

type Activity struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"userId" gorm:"not null;index"`
	Title            string         `json:"title" gorm:"not null;size:255"`
	Description      string         `json:"description" gorm:"type:text"`
	CompletionAmount int            `json:"completionAmount" gorm:"not null;default:1"`
	Frequency        Frequency      `json:"frequency" gorm:"not null"`
	DayFrequency     string         `json:"dayFrequency,omitempty" gorm:"type:text"` // JSON array of days for weekly activities
	IsActive         bool           `json:"isActive" gorm:"default:true"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (a *Activity) ToResponse() *dto.ActivityResponse {
	return &dto.ActivityResponse{
		ID:               a.ID,
		UserID:           a.UserID,
		Title:            a.Title,
		Description:      a.Description,
		CompletionAmount: a.CompletionAmount,
		Frequency:        string(a.Frequency),
		DayFrequency:     a.DayFrequency,
		IsActive:         a.IsActive,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}
