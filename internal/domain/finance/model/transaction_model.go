package model

import (
	"time"

	"life-tracker-backend/internal/domain/finance/dto"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	Date        time.Time            `bson:"date" json:"date"`
	CreatedAt   time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time            `bson:"updatedAt" json:"updatedAt"`
	Type        TransactionType      `bson:"type" json:"type"`
	Frequency   TransactionFrequency `bson:"frequency" json:"frequency"`
	Description string               `bson:"description,omitempty" json:"description,omitempty"`
	UserID      uint                 `bson:"userId" json:"userId"`
	Amount      float64              `bson:"amount" json:"amount"`
	CategoryID  uint                 `bson:"categoryId" json:"categoryId"`
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
}

func (t *Transaction) ToResponse(categoryName string) *dto.TransactionResponse {
	return &dto.TransactionResponse{
		ID:           t.ID.Hex(),
		UserID:       t.UserID,
		Type:         string(t.Type),
		Frequency:    string(t.Frequency),
		Amount:       t.Amount,
		CategoryID:   t.CategoryID,
		CategoryName: categoryName,
		Description:  t.Description,
		Date:         t.Date,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}
