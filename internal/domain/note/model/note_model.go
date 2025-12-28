package model

import (
	"time"

	"life-tracker-backend/internal/domain/note/dto"

	"gorm.io/gorm"
)

type Note struct {
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Title     string         `json:"title" gorm:"not null;size:255"`
	Content   string         `json:"content" gorm:"type:text"`
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"userId" gorm:"not null;index"`
}

func (n *Note) ToResponse() *dto.NoteResponse {
	return &dto.NoteResponse{
		ID:        n.ID,
		UserID:    n.UserID,
		Title:     n.Title,
		Content:   n.Content,
		CreatedAt: n.CreatedAt,
		EditedAt:  n.UpdatedAt,
	}
}
