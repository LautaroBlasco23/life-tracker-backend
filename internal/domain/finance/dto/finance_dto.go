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
	Date        time.Time `json:"date,omitempty"`
	Type        string    `json:"type" binding:"required,oneof=income outcome"`
	Frequency   string    `json:"frequency" binding:"required,oneof=fixed variable"`
	Description string    `json:"description,omitempty" binding:"max=500"`
	Amount      float64   `json:"amount" binding:"required,gt=0"`
	CategoryID  uint      `json:"categoryId" binding:"required"`
}

type UpdateTransactionRequest struct {
	Type        *string    `json:"type,omitempty" binding:"omitempty,oneof=income outcome"`
	Frequency   *string    `json:"frequency,omitempty" binding:"omitempty,oneof=fixed variable"`
	Amount      *float64   `json:"amount,omitempty" binding:"omitempty,gt=0"`
	CategoryID  *uint      `json:"categoryId,omitempty"`
	Description *string    `json:"description,omitempty" binding:"omitempty,max=500"`
	Date        *time.Time `json:"date,omitempty"`
}

type TransactionResponse struct {
	Date         time.Time `json:"date"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Frequency    string    `json:"frequency"`
	CategoryName string    `json:"categoryName"`
	Description  string    `json:"description,omitempty"`
	UserID       uint      `json:"userId"`
	Amount       float64   `json:"amount"`
	CategoryID   uint      `json:"categoryId"`
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
