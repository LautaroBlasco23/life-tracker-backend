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

type TransactionFrequency string

const (
	TransactionFrequencyFixed    TransactionFrequency = "fixed"
	TransactionFrequencyVariable TransactionFrequency = "variable"
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

func (f *TransactionFrequency) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan type %T into TransactionFrequency", value)
	}
	*f = TransactionFrequency(str)
	return nil
}

func (f TransactionFrequency) Value() (driver.Value, error) {
	return string(f), nil
}

type Category struct {
	CreatedAt        time.Time            `json:"createdAt"`
	UpdatedAt        time.Time            `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt       `json:"-" gorm:"index"`
	Name             string               `json:"name" gorm:"not null;size:100;uniqueIndex"`
	Type             TransactionType      `json:"type" gorm:"not null"`
	Icon             string               `json:"icon,omitempty" gorm:"size:50"`
	ApplicableToFreq TransactionFrequency `json:"applicableToFreq" gorm:"not null;size:20;default:variable"`
	ID               uint                 `json:"id" gorm:"primaryKey"`
}

var SystemCategories = []Category{
	// INCOME
	{Name: "Primary Income", Type: TransactionTypeIncome, Icon: "💰", ApplicableToFreq: TransactionFrequencyFixed},
	{Name: "Side Income", Type: TransactionTypeIncome, Icon: "💼", ApplicableToFreq: TransactionFrequencyVariable},
	{Name: "Investments", Type: TransactionTypeIncome, Icon: "📈", ApplicableToFreq: TransactionFrequencyVariable},
	{Name: "Other Income", Type: TransactionTypeIncome, Icon: "🎁", ApplicableToFreq: TransactionFrequencyVariable},

	// EXPENSE
	{Name: "Housing & Utilities", Type: TransactionTypeOutcome, Icon: "🏠", ApplicableToFreq: TransactionFrequencyFixed},
	{Name: "Food & Groceries", Type: TransactionTypeOutcome, Icon: "🍽️", ApplicableToFreq: TransactionFrequencyVariable},
	{Name: "Transportation", Type: TransactionTypeOutcome, Icon: "🚗", ApplicableToFreq: TransactionFrequencyVariable},
	{Name: "Health & Fitness", Type: TransactionTypeOutcome, Icon: "🏥", ApplicableToFreq: TransactionFrequencyVariable},
	{Name: "Education", Type: TransactionTypeOutcome, Icon: "📚", ApplicableToFreq: TransactionFrequencyVariable},
	{Name: "Entertainment & Travel", Type: TransactionTypeOutcome, Icon: "🎉", ApplicableToFreq: TransactionFrequencyVariable},
	{Name: "Taxes & Fees", Type: TransactionTypeOutcome, Icon: "📋", ApplicableToFreq: TransactionFrequencyFixed},
	{Name: "Pets & Misc", Type: TransactionTypeOutcome, Icon: "🐾", ApplicableToFreq: TransactionFrequencyVariable},
}
