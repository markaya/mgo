package models

import "database/sql/driver"

type TransactionType int

const (
	Income TransactionType = iota
	Expense
	TransferIn
	TransferOut
)

var transactionTypeName = map[TransactionType]string{
	Income:      "IN",
	Expense:     "EX",
	TransferIn:  "TIN",
	TransferOut: "TOUT",
}

var stringToTransactionType = map[string]TransactionType{
	"IN":   Income,
	"EX":   Expense,
	"TIN":  TransferIn,
	"TOUT": TransferOut,
}

func GetTransactionTypeFromString(s string) (TransactionType, bool) {
	v, b := stringToTransactionType[s]
	return v, b
}

func (t TransactionType) String() string {
	return transactionTypeName[t]
}

func (t *TransactionType) Scan(value any) error {
	*t = TransactionType(value.(int64))
	return nil
}

func (t TransactionType) Value() (driver.Value, error) {
	return int64(t), nil
}
