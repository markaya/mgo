package models

import "github.com/markaya/meinappf/internal/validator"

type TransactionCreateForm struct {
	UserId          int
	AccountId       int
	Date            string
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
	Date       string
	validator.Validator
}
