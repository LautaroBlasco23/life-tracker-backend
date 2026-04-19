package model

import (
	"time"

	"life-tracker-backend/internal/domain/screentime/dto"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WebVisit struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    uint               `bson:"userId" json:"userId"`
	Domain    string             `bson:"domain" json:"domain"`
	VisitedAt time.Time          `bson:"visitedAt" json:"visitedAt"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

func (v *WebVisit) ToResponse() *dto.VisitResponse {
	return &dto.VisitResponse{
		ID:        v.ID.Hex(),
		Domain:    v.Domain,
		VisitedAt: v.VisitedAt,
		CreatedAt: v.CreatedAt,
	}
}
