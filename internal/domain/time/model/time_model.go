package model

import (
	"time"

	"life-tracker-backend/internal/domain/time/dto"

	"gorm.io/gorm"
)

type TimeRecord struct {
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
	Category        string         `json:"category" gorm:"not null;size:100"`
	Description     string         `json:"description" gorm:"type:text"`
	ID              uint           `json:"id" gorm:"primaryKey"`
	UserID          uint           `json:"userId" gorm:"not null;index"`
	DurationMinutes int            `json:"durationMinutes" gorm:"not null"`
}

func (t *TimeRecord) ToResponse() *dto.TimeRecordResponse {
	return &dto.TimeRecordResponse{
		ID:              t.ID,
		Category:        t.Category,
		Description:     t.Description,
		DurationMinutes: t.DurationMinutes,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}
