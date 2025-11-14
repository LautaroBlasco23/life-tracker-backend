package model

import (
	"database/sql/driver"
	"fmt"
	"time"

	"life-tracker-backend/internal/domain/activity/dto"

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
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into Frequency", value)
	}
	*f = Frequency(str)
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

type DayTime string

const (
	DayTimeMorning   DayTime = "morning"
	DayTimeAfternoon DayTime = "afternoon"
	DayTimeEvening   DayTime = "evening"
)

func (dt *DayTime) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into DayTime", value)
	}
	*dt = DayTime(str)
	return nil
}

func (dt DayTime) Value() (driver.Value, error) {
	return string(dt), nil
}

type Activity struct {
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
	Title            string         `json:"title" gorm:"not null;size:255"`
	Description      string         `json:"description" gorm:"type:text"`
	Frequency        Frequency      `json:"frequency" gorm:"not null"`
	DayFrequency     string         `json:"dayFrequency,omitempty" gorm:"type:text"`
	DayTime          DayTime        `json:"dayTime" gorm:"not null;default:'morning'"`
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"userId" gorm:"not null;index"`
	CompletionAmount int            `json:"completionAmount" gorm:"not null;default:1"`
	IsActive         bool           `json:"isActive" gorm:"not null;default:false"`
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
		DayTime:          string(a.DayTime),
		IsActive:         a.IsActive,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
		TodayCompletions: 0,
		IsCompletedToday: false,
	}
}

func (a *Activity) ToResponseWithCompletions(todayCompletions int) *dto.ActivityResponse {
	response := a.ToResponse()
	response.TodayCompletions = todayCompletions
	response.IsCompletedToday = todayCompletions >= a.CompletionAmount
	return response
}
