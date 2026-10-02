package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
)

// UserRepository persists operator accounts.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a UserRepository backed by db.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user using tx.
func (r *UserRepository) Create(ctx context.Context, tx DBTX, u *model.User) error {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES (?, ?, ?, ?)`,
		u.Name, u.Email, u.PasswordHash, u.Role)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = id
	return nil
}

// FindByEmail returns the user with the given email, or ErrNotFound.
func (r *UserRepository) FindByEmail(ctx context.Context, q DBTX, email string) (*model.User, error) {
	var u model.User
	err := q.QueryRowContext(ctx,
		`SELECT id, name, email, password_hash, role, created_at, updated_at
		 FROM users WHERE email = ?`, email).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

// FindByID returns the user with the given id, or ErrNotFound.
func (r *UserRepository) FindByID(ctx context.Context, q DBTX, id int64) (*model.User, error) {
	var u model.User
	err := q.QueryRowContext(ctx,
		`SELECT id, name, email, password_hash, role, created_at, updated_at
		 FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}
