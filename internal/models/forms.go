package models

import (
	"time"

	"github.com/markaya/meinappf/internal/validator"
)

type TransactionCreateForm struct {
	UserId          int
	AccountId       int
	Date            time.Time
	Amount          float64
	Category        string
	Description     string
	Currency        int
	TransactionType int
	validator.Validator
}

type TransferCreateForm struct {
	FromAcc    Account
	FromAmount float64
	ToAcc      Account
	ToAmount   float64
	Date       time.Time
	validator.Validator
}
