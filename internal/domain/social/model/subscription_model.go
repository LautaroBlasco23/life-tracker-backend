package model

import (
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type SourceType string

const (
	SourceTypeActivity SourceType = "activity"
	SourceTypeRoutine  SourceType = "routine"
)

func (s *SourceType) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into SourceType", value)
	}
	*s = SourceType(str)
	return nil
}

func (s SourceType) Value() (driver.Value, error) {
	return string(s), nil
}

type Subscription struct {
	CreatedAt        time.Time      `json:"createdAt"`
	DeletedAt        gorm.DeletedAt `json:"-"                gorm:"index"`
	SourceType       SourceType     `json:"sourceType"       gorm:"not null;size:20"`
	SubscriberUserID uint           `json:"subscriberUserId" gorm:"not null;index"`
	SourceUserID     uint           `json:"sourceUserId"     gorm:"not null"`
	SourceID         uint           `json:"sourceId"         gorm:"not null"`
	ClonedID         uint           `json:"clonedId"         gorm:"not null"`
	ID               uint           `json:"id"               gorm:"primaryKey"`
}
