package models

import "database/sql/driver"

type Currency int

const (
	SerbianDinar Currency = iota
	Euro
)

var currencyTypeName = map[Currency]string{
	SerbianDinar: "RSD",
	Euro:         "EUR",
}

var stringToCurrencyType = map[string]Currency{
	"RSD": SerbianDinar,
	"EUR": Euro,
}

func GetCurrencyFromString(s string) (Currency, bool) {
	v, b := stringToCurrencyType[s]
	return v, b
}

func (c Currency) String() string {
	return currencyTypeName[c]
}

func (c *Currency) Scan(value any) error {
	*c = Currency(value.(int64))
	return nil
}

func (c Currency) Value() (driver.Value, error) {
	return int64(c), nil
}
