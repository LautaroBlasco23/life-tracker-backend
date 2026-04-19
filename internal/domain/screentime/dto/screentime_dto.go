package dto

import "time"

type RecordVisitRequest struct {
	Domain    string    `json:"domain" binding:"required"`
	VisitedAt time.Time `json:"visitedAt" binding:"required"`
}

type VisitResponse struct {
	ID        string    `json:"id"`
	Domain    string    `json:"domain"`
	VisitedAt time.Time `json:"visitedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type DomainStat struct {
	Domain string `json:"domain" bson:"_id"`
	Count  int64  `json:"count" bson:"count"`
}
