package model

import (
	"time"

	"life-tracker-backend/internal/domain/finance/dto"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        uint               `bson:"userId" json:"userId"`
	Type          TransactionType    `bson:"type" json:"type"`
	Amount        float64            `bson:"amount" json:"amount"`
	CategoryID    uint               `bson:"categoryId" json:"categoryId"`
	SubcategoryID uint               `bson:"subcategoryId" json:"subcategoryId"`
	Description   string             `bson:"description,omitempty" json:"description,omitempty"`
	Date          time.Time          `bson:"date" json:"date"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}

func (t *Transaction) ToResponse(categoryName, subcategoryName string) *dto.TransactionResponse {
	return &dto.TransactionResponse{
		ID:              t.ID.Hex(),
		UserID:          t.UserID,
		Type:            string(t.Type),
		Amount:          t.Amount,
		CategoryID:      t.CategoryID,
		CategoryName:    categoryName,
		SubcategoryID:   t.SubcategoryID,
		SubcategoryName: subcategoryName,
		Description:     t.Description,
		Date:            t.Date,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}
