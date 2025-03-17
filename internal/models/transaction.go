package models

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/mattn/go-sqlite3"
)

type TransactionsModelInterface interface {
	Insert(tf TransactionCreateForm, newBalance float64) (int, error)
	InsertTransfer(tf TransferCreateForm) error
	Get(id int) (*Transaction, error)
	GetAll(userId int) ([]*Transaction, error)
	GetByDate(userId int, startDate, endDate time.Time) ([]*Transaction, error)
	GetByType(userId int, tt TransactionType) ([]*Transaction, error)
	GetByDateAndType(userId int, tt TransactionType, startDate, endDate time.Time) ([]*Transaction, error)
	GetLatest(userId, limit int, tt TransactionType) ([]*Transaction, error)
}

type Transaction struct {
	ID              int
	AccountID       int
	UserID          int
	Date            time.Time
	Amount          float64
	Currency        Currency
	Category        string
	Description     string
	TransactionType TransactionType
}

func NewRebalance(account Account, balanceDiff float64) TransactionCreateForm {
	var transactionType TransactionType
	if balanceDiff > 0 {
		transactionType = RebalanceOut
	} else {
		transactionType = RebalanceIn
	}
	return TransactionCreateForm{
		UserId:          account.UserId,
		AccountId:       account.ID,
		Date:            time.Now().Format("2006-01-02"),
		Amount:          math.Abs(balanceDiff),
		Currency:        int(account.Currency),
		Category:        "rebalance",
		Description:     "rebalance of account",
		TransactionType: int(transactionType),
	}
}

func (a Transaction) DisplayAmount() string {
	return fmt.Sprintf("%.3f %s", a.Amount, a.Currency)
}

func (a Transaction) DisplayDate() string {
	return a.Date.Format("02-01-2006")
}

type TransactionModel struct {
	DB *sql.DB
}

func (m *TransactionModel) InsertTransfer(tf TransferCreateForm) error {
	stmt1 := `
	INSERT INTO transactions (account_id, user_id, date, amount, currency, category, description, transaction_type) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?);`

	stmt2 := `UPDATE accounts SET balance = ? WHERE id = ?;`
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			// HACK: What if there is an error requiring rollback but rollback fails?
			_ = tx.Rollback()
		}
	}()

	desc := fmt.Sprintf("[T] from %s to %s", tf.FromAcc.AccountName, tf.ToAcc.AccountName)
	// FIXME: make date from DATE string
	_, err = tx.Exec(stmt1, tf.FromAcc.ID, tf.FromAcc.UserId, tf.Date, tf.FromAmount, tf.FromAcc.Currency, "transfer", desc, TransferIn)

	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey {
				return ErrAccountDoesNotExist
			}
		}
		return err
	}

	newBalance := tf.FromAcc.Balance - tf.FromAmount
	_, err = tx.Exec(stmt2, newBalance, tf.FromAcc.ID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(stmt1, tf.ToAcc.ID, tf.ToAcc.UserId, tf.Date, tf.ToAmount, tf.ToAcc.Currency, "transfer", desc, TransferOut)

	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey {
				return ErrAccountDoesNotExist
			}
		}
		return err
	}

	newBalance = tf.ToAcc.Balance + tf.ToAmount
	_, err = tx.Exec(stmt2, newBalance, tf.ToAcc.ID)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (m *TransactionModel) Insert(tf TransactionCreateForm, newBalance float64) (int, error) {
	stmt1 := `
	INSERT INTO transactions (account_id, user_id, date, amount, currency, category, description, transaction_type) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?);`

	stmt2 := `UPDATE accounts SET balance = ? WHERE id = ?;`

	tx, err := m.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			// HACK: What if there is an error requiring rollback but rollback fails?
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Exec(stmt1, tf.AccountId, tf.UserId, tf.Date, tf.Amount, Currency(tf.Currency), tf.Category, tf.Description, TransactionType(tf.TransactionType))

	if err != nil {
		sqliteErr, ok := err.(sqlite3.Error)
		if ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey {
				return 0, ErrAccountDoesNotExist
			}
		}
		return 0, err
	}

	result, err := tx.Exec(stmt2, newBalance, tf.AccountId)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("failed to insert transaction")
	}

	return int(id), nil
}

func (m *TransactionModel) Get(id int) (*Transaction, error) {
	stmt := `
	SELECT id, account_id, user_id, date, amount, currency, category, description, transaction_type
	FROM transactions
	WHERE transaction_id = ?;`

	row := m.DB.QueryRow(stmt, id)
	t := &Transaction{}

	err := row.Scan(&t.ID, &t.AccountID, &t.UserID, &t.Date, &t.Amount, &t.Currency, &t.Category, &t.Description, &t.TransactionType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}

	return t, nil
}

func (m *TransactionModel) GetAll(userId int) ([]*Transaction, error) {
	stmt := `
	SELECT id, account_id, user_id, date, amount, currency, category, description, transaction_type
	FROM transactions
	WHERE user_id = ?;`

	rows, err := m.DB.Query(stmt, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []*Transaction{}

	for rows.Next() {
		t := &Transaction{}
		err := rows.Scan(&t.ID, &t.AccountID, &t.UserID, &t.Date, &t.Amount, &t.Currency, &t.Category, &t.Description, &t.TransactionType)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}
func (m *TransactionModel) GetByDateAndType(userId int, tt TransactionType, startDate, endDate time.Time) ([]*Transaction, error) {
	stmt := `
	SELECT id, account_id, user_id, date, amount, currency, category, description, transaction_type
	FROM transactions
	WHERE user_id = ?
	AND date between ? and ?
	AND transaction_type = ?
	ORDER BY date DESC, id DESC;`

	rows, err := m.DB.Query(stmt, userId, startDate, endDate, tt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []*Transaction{}

	for rows.Next() {
		t := &Transaction{}
		err := rows.Scan(&t.ID, &t.AccountID, &t.UserID, &t.Date, &t.Amount, &t.Currency, &t.Category, &t.Description, &t.TransactionType)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}
func (m *TransactionModel) GetByDate(userId int, startDate, endDate time.Time) ([]*Transaction, error) {
	stmt := `
	SELECT id, account_id, user_id, date, amount, currency, category, description, transaction_type
	FROM transactions
	WHERE user_id = ?
	AND date between ? and ?
	ORDER BY date DESC, id DESC;`

	rows, err := m.DB.Query(stmt, userId, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []*Transaction{}

	for rows.Next() {
		t := &Transaction{}
		err := rows.Scan(&t.ID, &t.AccountID, &t.UserID, &t.Date, &t.Amount, &t.Currency, &t.Category, &t.Description, &t.TransactionType)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (m *TransactionModel) GetByType(userId int, tt TransactionType) ([]*Transaction, error) {
	stmt := `
	SELECT id, account_id, user_id, date, amount, currency, category, description, transaction_type
	FROM transactions
	WHERE user_id = ?
	AND transaction_type = ?;`

	rows, err := m.DB.Query(stmt, userId, tt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []*Transaction{}

	for rows.Next() {
		t := &Transaction{}
		err := rows.Scan(&t.ID, &t.AccountID, &t.UserID, &t.Date, &t.Amount, &t.Currency, &t.Category, &t.Description, &t.TransactionType)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (m *TransactionModel) GetLatest(userId, limit int, tt TransactionType) ([]*Transaction, error) {
	stmt := `
	SELECT id, account_id, user_id, date, amount, currency, category, description, transaction_type
	FROM transactions
	WHERE user_id = ?
	AND transaction_type = ?
	ORDER BY date DESC, id DESC
	LIMIT ?;`
	//  AND date between ? and ?

	rows, err := m.DB.Query(stmt, userId, tt, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []*Transaction{}

	for rows.Next() {
		t := &Transaction{}
		err := rows.Scan(&t.ID, &t.AccountID, &t.UserID, &t.Date, &t.Amount, &t.Currency, &t.Category, &t.Description, &t.TransactionType)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}
