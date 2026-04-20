package model

import (
	"time"

	"gorm.io/gorm"
)

type Auth struct {
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	Email        string         `json:"email" gorm:"uniqueIndex;not null"`
	Username     string         `json:"username" gorm:"uniqueIndex;not null;size:64"`
	PasswordHash string         `json:"-" gorm:"not null"`
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"userId" gorm:"not null"`
}
