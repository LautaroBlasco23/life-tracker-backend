package dto

import "time"

type CategoryResponse struct {
	ID            uint                  `json:"id"`
	Name          string                `json:"name"`
	Type          string                `json:"type"`
	Icon          string                `json:"icon,omitempty"`
	Subcategories []SubcategoryResponse `json:"subcategories,omitempty"`
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`
}

type SubcategoryResponse struct {
	ID         uint      `json:"id"`
	CategoryID uint      `json:"categoryId"`
	Name       string    `json:"name"`
	Icon       string    `json:"icon,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type CreateTransactionRequest struct {
	Type          string    `json:"type" binding:"required,oneof=income outcome"`
	Amount        float64   `json:"amount" binding:"required,gt=0"`
	CategoryID    uint      `json:"categoryId" binding:"required"`
	SubcategoryID uint      `json:"subcategoryId" binding:"required"`
	Description   string    `json:"description,omitempty" binding:"max=500"`
	Date          time.Time `json:"date,omitempty"`
}

type UpdateTransactionRequest struct {
	Type          *string    `json:"type,omitempty" binding:"omitempty,oneof=income outcome"`
	Amount        *float64   `json:"amount,omitempty" binding:"omitempty,gt=0"`
	CategoryID    *uint      `json:"categoryId,omitempty"`
	SubcategoryID *uint      `json:"subcategoryId,omitempty"`
	Description   *string    `json:"description,omitempty" binding:"omitempty,max=500"`
	Date          *time.Time `json:"date,omitempty"`
}

type TransactionResponse struct {
	ID              string    `json:"id"`
	UserID          uint      `json:"userId"`
	Type            string    `json:"type"`
	Amount          float64   `json:"amount"`
	CategoryID      uint      `json:"categoryId"`
	CategoryName    string    `json:"categoryName"`
	SubcategoryID   uint      `json:"subcategoryId"`
	SubcategoryName string    `json:"subcategoryName"`
	Description     string    `json:"description,omitempty"`
	Date            time.Time `json:"date"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type FinanceSummaryResponse struct {
	TotalIncome       float64           `json:"totalIncome"`
	TotalOutcome      float64           `json:"totalOutcome"`
	Balance           float64           `json:"balance"`
	IncomeByCategory  []CategorySummary `json:"incomeByCategory"`
	OutcomeByCategory []CategorySummary `json:"outcomeByCategory"`
	Period            string            `json:"period"`
}

type CategorySummary struct {
	CategoryID   uint    `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
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
