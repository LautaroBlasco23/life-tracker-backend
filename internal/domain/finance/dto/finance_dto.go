package dto

import "time"

type CategoryResponse struct {
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	Icon             string    `json:"icon,omitempty"`
	ApplicableToFreq string    `json:"applicableToFreq"`
	ID               uint      `json:"id"`
}

type CreateTransactionRequest struct {
	Date             time.Time `json:"date,omitempty"`
	Type             string    `json:"type" binding:"required,oneof=income outcome"`
	Frequency        string    `json:"frequency" binding:"required,oneof=fixed variable"`
	PaymentFrequency string    `json:"paymentFrequency,omitempty" binding:"omitempty,oneof=monthly bimonthly yearly"`
	Description      string    `json:"description,omitempty" binding:"max=500"`
	Amount           float64   `json:"amount" binding:"gte=0"`
	CategoryID       uint      `json:"categoryId" binding:"required"`
}

type UpdateTransactionRequest struct {
	Type             *string    `json:"type,omitempty" binding:"omitempty,oneof=income outcome"`
	Frequency        *string    `json:"frequency,omitempty" binding:"omitempty,oneof=fixed variable"`
	PaymentFrequency *string    `json:"paymentFrequency,omitempty" binding:"omitempty,oneof=monthly bimonthly yearly"`
	Amount           *float64   `json:"amount,omitempty" binding:"omitempty,gte=0"`
	CategoryID       *uint      `json:"categoryId,omitempty"`
	Description      *string    `json:"description,omitempty" binding:"omitempty,max=500"`
	Date             *time.Time `json:"date,omitempty"`
}

type TransactionResponse struct {
	Date             time.Time `json:"date"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	Frequency        string    `json:"frequency"`
	PaymentFrequency string    `json:"paymentFrequency,omitempty"`
	CategoryName     string    `json:"categoryName"`
	Description      string    `json:"description,omitempty"`
	UserID           uint      `json:"userId"`
	Amount           float64   `json:"amount"`
	CategoryID       uint      `json:"categoryId"`
}

type FixedTransactionResponse struct {
	CreatedAt            time.Time        `json:"createdAt"`
	UpdatedAt            time.Time        `json:"updatedAt"`
	ID                   string           `json:"id"`
	Type                 string           `json:"type"`
	PaymentFrequency     string           `json:"paymentFrequency"`
	CategoryName         string           `json:"categoryName"`
	Description          string           `json:"description,omitempty"`
	TotalPaid            float64          `json:"totalPaid"`
	Payments             []PaymentSummary `json:"payments,omitempty"`
	CurrentPeriodPayment *PaymentSummary  `json:"currentPeriodPayment,omitempty"`
	CategoryID           uint             `json:"categoryId"`
}

type PaymentSummary struct {
	ID     string    `json:"id"`
	Amount float64   `json:"amount"`
	Date   time.Time `json:"date"`
}

type CreatePaymentRequest struct {
	TransactionID string    `json:"transactionId" binding:"required"`
	Amount        float64   `json:"amount" binding:"required,gt=0"`
	Date          time.Time `json:"date,omitempty"`
}

type PaymentResponse struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transactionId"`
	Amount        float64   `json:"amount"`
	Date          time.Time `json:"date"`
	CreatedAt     time.Time `json:"createdAt"`
}

type FinanceSummaryResponse struct {
	Period            string            `json:"period"`
	IncomeByCategory  []CategorySummary `json:"incomeByCategory"`
	OutcomeByCategory []CategorySummary `json:"outcomeByCategory"`
	TotalIncome       float64           `json:"totalIncome"`
	TotalOutcome      float64           `json:"totalOutcome"`
	Balance           float64           `json:"balance"`
}

type CategorySummary struct {
	CategoryName string  `json:"categoryName"`
	CategoryID   uint    `json:"categoryId"`
	Total        float64 `json:"total"`
	Percentage   float64 `json:"percentage"`
	Count        int64   `json:"count"`
}

type MonthlyStatsResponse struct {
	Month            string  `json:"month"`
	Year             int     `json:"year"`
	TotalIncome      float64 `json:"totalIncome"`
	TotalOutcome     float64 `json:"totalOutcome"`
	Balance          float64 `json:"balance"`
	TransactionCount int64   `json:"transactionCount"`
}
