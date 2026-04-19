package model

import (
	"time"

	"life-tracker-backend/internal/domain/finance/dto"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	ID               primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID           uint                 `bson:"userId" json:"userId"`
	Type             TransactionType      `bson:"type" json:"type"`
	Frequency        TransactionFrequency `bson:"frequency" json:"frequency"`
	PaymentFrequency PaymentFrequency     `bson:"paymentFrequency,omitempty" json:"paymentFrequency,omitempty"`
	Amount           float64              `bson:"amount" json:"amount"`
	Category         string               `bson:"category" json:"category"`
	Description      string               `bson:"description,omitempty" json:"description,omitempty"`
	Date             time.Time            `bson:"date" json:"date"`
	CreatedAt        time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time            `bson:"updatedAt" json:"updatedAt"`
}

func (t *Transaction) IsFixed() bool {
	return t.Frequency == TransactionFrequencyFixed
}

func (t *Transaction) ToResponse() *dto.TransactionResponse {
	return &dto.TransactionResponse{
		ID:               t.ID.Hex(),
		UserID:           t.UserID,
		Type:             string(t.Type),
		Frequency:        string(t.Frequency),
		PaymentFrequency: string(t.PaymentFrequency),
		Amount:           t.Amount,
		CategoryName:     t.Category,
		Description:      t.Description,
		Date:             t.Date,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

func (t *Transaction) ToFixedResponse(totalPaid float64, payments []dto.PaymentSummary, currentPeriodPayment *dto.PaymentSummary) *dto.FixedTransactionResponse {
	return &dto.FixedTransactionResponse{
		ID:                   t.ID.Hex(),
		Type:                 string(t.Type),
		Frequency:            string(t.Frequency),
		PaymentFrequency:     string(t.PaymentFrequency),
		CategoryName:         t.Category,
		Description:          t.Description,
		TotalPaid:            totalPaid,
		Payments:             payments,
		CurrentPeriodPayment: currentPeriodPayment,
		CreatedAt:            t.CreatedAt,
		UpdatedAt:            t.UpdatedAt,
	}
}
