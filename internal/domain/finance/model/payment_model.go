package model

import (
	"time"

	"life-tracker-backend/internal/domain/finance/dto"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentFrequency string

const (
	PaymentFrequencyMonthly   PaymentFrequency = "monthly"
	PaymentFrequencyBimonthly PaymentFrequency = "bimonthly"
	PaymentFrequencyYearly    PaymentFrequency = "yearly"
)

type Payment struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TransactionID primitive.ObjectID `bson:"transactionId" json:"transactionId"`
	UserID        uint               `bson:"userId" json:"userId"`
	Amount        float64            `bson:"amount" json:"amount"`
	Date          time.Time          `bson:"date" json:"date"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
}

func (p *Payment) ToResponse() *dto.PaymentResponse {
	return &dto.PaymentResponse{
		ID:            p.ID.Hex(),
		TransactionID: p.TransactionID.Hex(),
		Amount:        p.Amount,
		Date:          p.Date,
		CreatedAt:     p.CreatedAt,
	}
}

func (p *Payment) ToSummary() dto.PaymentSummary {
	return dto.PaymentSummary{
		ID:     p.ID.Hex(),
		Amount: p.Amount,
		Date:   p.Date,
	}
}

func (p *Payment) GetPeriodKey(frequency PaymentFrequency) string {
	year, month, _ := p.Date.Date()

	switch frequency {
	case PaymentFrequencyMonthly:
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	case PaymentFrequencyBimonthly:
		bimonthlyPeriod := (int(month) - 1) / 2
		return time.Date(year, time.Month(bimonthlyPeriod*2+1), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	case PaymentFrequencyYearly:
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006")
	default:
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	}
}

func GetCurrentPeriodKey(frequency PaymentFrequency) string {
	now := time.Now()
	year, month, _ := now.Date()

	switch frequency {
	case PaymentFrequencyMonthly:
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	case PaymentFrequencyBimonthly:
		bimonthlyPeriod := (int(month) - 1) / 2
		return time.Date(year, time.Month(bimonthlyPeriod*2+1), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	case PaymentFrequencyYearly:
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006")
	default:
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
	}
}
