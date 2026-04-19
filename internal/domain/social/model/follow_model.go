package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type FollowStatus string

const (
	FollowStatusPending  FollowStatus = "pending"
	FollowStatusAccepted FollowStatus = "accepted"
)

func (f *FollowStatus) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into FollowStatus", value)
	}
	*f = FollowStatus(str)
	return nil
}

func (f FollowStatus) Value() (driver.Value, error) {
	return string(f), nil
}

// Follow has composite PK (follower_id, followee_id) — no gorm.Model, no deleted_at.
type Follow struct {
	CreatedAt  time.Time    `json:"createdAt"`
	AcceptedAt *time.Time   `json:"acceptedAt,omitempty"`
	Status     FollowStatus `json:"status"     gorm:"not null;size:20"`
	FollowerID uint         `json:"followerId" gorm:"primaryKey;not null"`
	FolloweeID uint         `json:"followeeId" gorm:"primaryKey;not null"`
}

func (Follow) TableName() string { return "follows" }
