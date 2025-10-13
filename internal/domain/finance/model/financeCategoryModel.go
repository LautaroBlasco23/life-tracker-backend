package model

import (
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypeIncome  TransactionType = "income"
	TransactionTypeOutcome TransactionType = "outcome"
)

func (t *TransactionType) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into TransactionType", value)
	}
	*t = TransactionType(str)
	return nil
}

func (t TransactionType) Value() (driver.Value, error) {
	return string(t), nil
}

type Category struct {
	ID        uint            `json:"id" gorm:"primaryKey"`
	Name      string          `json:"name" gorm:"not null;size:100;uniqueIndex"`
	Type      TransactionType `json:"type" gorm:"not null"`
	Icon      string          `json:"icon,omitempty" gorm:"size:50"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	DeletedAt gorm.DeletedAt  `json:"-" gorm:"index"`
}

type Subcategory struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	CategoryID uint           `json:"categoryId" gorm:"not null;index"`
	Name       string         `json:"name" gorm:"not null;size:100"`
	Icon       string         `json:"icon,omitempty" gorm:"size:50"`
	Category   *Category      `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

var SystemCategories = []Category{
	{Name: "Salary", Type: TransactionTypeIncome, Icon: "💰"},
	{Name: "Investments", Type: TransactionTypeIncome, Icon: "📈"},
	{Name: "Freelance", Type: TransactionTypeIncome, Icon: "💼"},
	{Name: "Refunds", Type: TransactionTypeIncome, Icon: "↩️"},
	{Name: "Other Incomes", Type: TransactionTypeIncome, Icon: "🎁"},
	{Name: "Services", Type: TransactionTypeOutcome, Icon: "🔧"},
	{Name: "Housing", Type: TransactionTypeOutcome, Icon: "🏠"},
	{Name: "Food", Type: TransactionTypeOutcome, Icon: "🍽️"},
	{Name: "Taxes", Type: TransactionTypeOutcome, Icon: "📋"},
	{Name: "Gym & Sports", Type: TransactionTypeOutcome, Icon: "⚽"},
	{Name: "Entertainment", Type: TransactionTypeOutcome, Icon: "🎉"},
	{Name: "Travel", Type: TransactionTypeOutcome, Icon: "✈️"},
	{Name: "Gifts & Events", Type: TransactionTypeOutcome, Icon: "🎁"},
	{Name: "Health", Type: TransactionTypeOutcome, Icon: "🏥"},
	{Name: "Education", Type: TransactionTypeOutcome, Icon: "📚"},
	{Name: "Transportation", Type: TransactionTypeOutcome, Icon: "🚗"},
	{Name: "Pets", Type: TransactionTypeOutcome, Icon: "🐾"},
	{Name: "Other Expenses", Type: TransactionTypeOutcome, Icon: "📦"},
}

var SystemSubcategories = map[string][]string{
	"Salary":         {"Monthly Wage", "Bonuses", "Commissions"},
	"Investments":    {"Dividends", "Crypto Gains", "Interests"},
	"Freelance":      {"Projects", "Side Jobs", "Consulting"},
	"Refunds":        {"Tax Refund", "Returned Purchases"},
	"Other Incomes":  {"Gifts", "One-time Payments"},
	"Services":       {"Internet", "Phone", "Subscriptions", "Streaming"},
	"Housing":        {"Rent", "Mortgage", "Insurance", "Maintenance"},
	"Food":           {"Groceries", "Dining", "Delivery"},
	"Taxes":          {"Property Tax", "Income Tax", "Bills"},
	"Gym & Sports":   {"Memberships", "Football", "Classes"},
	"Entertainment":  {"Movies", "Bars", "Concerts"},
	"Travel":         {"Tickets", "Hotels", "Transport"},
	"Gifts & Events": {"Birthdays", "Holidays"},
	"Health":         {"Medicines", "Doctor Visits", "Therapy"},
	"Education":      {"Courses", "Books", "Materials"},
	"Transportation": {"Fuel", "Parking", "Public Transport"},
	"Pets":           {"Food", "Vet", "Accessories"},
	"Other Expenses": {"Uncategorized", "One-off Costs"},
}
