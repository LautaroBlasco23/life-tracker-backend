package dto

import "time"

type CategoryResponse struct {
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`
	Name          string                `json:"name"`
	Type          string                `json:"type"`
	Icon          string                `json:"icon,omitempty"`
	Subcategories []SubcategoryResponse `json:"subcategories,omitempty"`
	ID            uint                  `json:"id"`
}

type SubcategoryResponse struct {
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Name       string    `json:"name"`
	Icon       string    `json:"icon,omitempty"`
	ID         uint      `json:"id"`
	CategoryID uint      `json:"categoryId"`
}

type CreateTransactionRequest struct {
	Date          time.Time `json:"date,omitempty"`
	Type          string    `json:"type" binding:"required,oneof=income outcome"`
	Description   string    `json:"description,omitempty" binding:"max=500"`
	Amount        float64   `json:"amount" binding:"required,gt=0"`
	CategoryID    uint      `json:"categoryId" binding:"required"`
	SubcategoryID uint      `json:"subcategoryId" binding:"required"`
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
	Date            time.Time `json:"date"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	CategoryName    string    `json:"categoryName"`
	SubcategoryName string    `json:"subcategoryName"`
	Description     string    `json:"description,omitempty"`
	UserID          uint      `json:"userId"`
	Amount          float64   `json:"amount"`
	CategoryID      uint      `json:"categoryId"`
	SubcategoryID   uint      `json:"subcategoryId"`
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
