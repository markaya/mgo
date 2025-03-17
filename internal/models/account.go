package models

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
)

type AccountModelInterface interface {
	Insert(userID int, accountName string, currency Currency) (int, error)
	Get(userId, id int) (*Account, error)
	GetAll(userId int) ([]*Account, error)
}

type Account struct {
	ID          int
	UserId      int
	AccountName string
	Balance     float64
	Currency    Currency
}

func (a Account) DisplayBalance() string {
	return fmt.Sprintf("%.2f %s", a.Balance, a.Currency)
}

type AccountModel struct {
	DB *sql.DB
}

func (m *AccountModel) Insert(userID int, accountName string, currency Currency) (int, error) {
	stmt := `INSERT INTO accounts (user_id, account_name, balance, currency) 
	VALUES (?, ?, ?, ?)`

	result, err := m.DB.Exec(stmt, userID, accountName, 0, currency)
	if err != nil {
		sqliteErr, b := err.(sqlite3.Error)
		if b {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				return 0, ErrDuplicateAccountName
			}
		}
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	if id == 0 {
		return 0, errors.New("failed to insert account")
	}

	return int(id), nil
}

func (m *AccountModel) Get(userId, id int) (*Account, error) {
	stmt := `SELECT id, user_id, account_name, balance, currency
	FROM accounts 
	WHERE user_id = ?
	AND id = ?
	`

	s := &Account{}

	row := m.DB.QueryRow(stmt, userId, id)
	err := row.Scan(&s.ID, &s.UserId, &s.AccountName, &s.Balance, &s.Currency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}
	return s, nil
}

func (m *AccountModel) GetAll(userId int) ([]*Account, error) {
	stmt := `SELECT id, user_id, account_name, balance, currency FROM accounts where user_id = ?`

	rows, err := m.DB.Query(stmt, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []*Account{}

	for rows.Next() {
		a := &Account{}
		err := rows.Scan(&a.ID, &a.UserId, &a.AccountName, &a.Balance, &a.Currency)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}
