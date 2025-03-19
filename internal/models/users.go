package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type UserModelInterface interface {
	Insert(name, email, password string) error
	Authenticate(email, password string) (int, error)
	Exist(id int) (bool, error)
	Get(id int) (*User, error)
	UpdatePassword(id int, password string) error
}

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Get(id int) (*User, error) {
	u := &User{}
	stmt := `SELECT id, name, email, created, hashed_password FROM users WHERE id = ?`

	err := m.DB.QueryRow(stmt, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.Created, &u.HashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}

	return u, nil
}

func HashPassword(password string) ([]byte, error) {
	// NOTE: Add secret string to password before hashing (Password + secret)
	// Keep secret as env variable
	// No need as bcrypt does salting itself
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 13)
	if err != nil {
		return nil, err
	}
	return hashedPassword, nil
}

func (m *UserModel) Insert(name, email, password string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	stmt := `INSERT INTO users (name, email, hashed_password, created)
	VALUES(?,?,?, datetime('now', 'utc'))`

	_, err = m.DB.Exec(stmt, name, email, string(hashedPassword))
	if err != nil {
		sqliteErr, b := err.(sqlite3.Error)
		if b {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	return nil
}

func (m *UserModel) UpdatePassword(id int, password string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}

	stmt := `UPDATE users SET hashed_password=? WHERE id=?`

	_, err = m.DB.Exec(stmt, string(hashedPassword), id)
	if err != nil {
		sqliteErr, b := err.(sqlite3.Error)
		if b {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	return nil
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	var id int
	var hashedPassword []byte

	stmt := "SELECT id, hashed_password FROM users where email = ?"

	err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	return id, nil
}

func (m *UserModel) Exist(id int) (bool, error) {
	var exists bool
	stmt := "SELECT EXISTS(SELECT true FROM users WHERE id=?)"
	err := m.DB.QueryRow(stmt, id).Scan(&exists)
	return exists, err
}
