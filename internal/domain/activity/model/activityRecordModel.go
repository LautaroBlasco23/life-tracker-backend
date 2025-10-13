package model

import (
	"time"

	"life-tracker-backend/internal/domain/activity/dto"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivityRecord represents a completion record in MongoDB
type ActivityRecord struct {
	CompletionDate time.Time          `bson:"completionDate" json:"completionDate"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
	Notes          string             `bson:"notes,omitempty" json:"notes,omitempty"`
	ActivityID     uint               `bson:"activityId" json:"activityId"`
	UserID         uint               `bson:"userId" json:"userId"`
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
}

func (ar *ActivityRecord) ToResponse() *dto.ActivityRecordResponse {
	return &dto.ActivityRecordResponse{
		ID:             ar.ID.Hex(),
		ActivityID:     ar.ActivityID,
		UserID:         ar.UserID,
		CompletionDate: ar.CompletionDate,
		Notes:          ar.Notes,
		CreatedAt:      ar.CreatedAt,
	}
}

// ActivityStats represents aggregated statistics for an activity
type ActivityStats struct {
	Title            string                        `bson:"title" json:"title"`
	RecentRecords    []*dto.ActivityRecordResponse `bson:"recentRecords" json:"recentRecords"`
	ActivityID       uint                          `bson:"activityId" json:"activityId"`
	TotalCompletions int64                         `bson:"totalCompletions" json:"totalCompletions"`
	CurrentStreak    int                           `bson:"currentStreak" json:"currentStreak"`
	LongestStreak    int                           `bson:"longestStreak" json:"longestStreak"`
	CompletionRate   float64                       `bson:"completionRate" json:"completionRate"`
}
