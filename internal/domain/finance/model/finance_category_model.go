package model

import (
	"database/sql/driver"
	"fmt"
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
