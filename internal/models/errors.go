package models

import (
	"errors"
)

var (
	ErrNoRecord = errors.New("models: no matching records found")

	ErrInvalidCredentials = errors.New("models: invalid credentials")

	ErrDuplicateEmail = errors.New("models: duplicate email")

	ErrDuplicateAccountName = errors.New("accounts: duplicate account_name per user")

	ErrAccountDoesNotExist = errors.New("transactions: user account does not exist")
)
