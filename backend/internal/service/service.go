// Package service implements the business logic of the API.
package service

import (
	"context"
	"database/sql"
	"net/mail"
	"strings"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

// Transactor runs fn inside a database transaction, committing on success
// and rolling back on error or panic.
type Transactor func(ctx context.Context, fn func(tx repository.DBTX) error) error

// DBTransactor adapts *sql.DB to Transactor for production use.
func DBTransactor(db *sql.DB) Transactor {
	return func(ctx context.Context, fn func(tx repository.DBTX) error) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() {
			if p := recover(); p != nil {
				_ = tx.Rollback()
				panic(p)
			}
		}()
		if err := fn(tx); err != nil {
			_ = tx.Rollback()
			return err
		}
		return tx.Commit()
	}
}

// ValidationError describes a single invalid input field.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements error.
func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func validEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
