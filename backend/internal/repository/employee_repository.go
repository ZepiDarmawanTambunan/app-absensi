package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
)

// EmployeeRepository persists employees.
type EmployeeRepository struct {
	db *sql.DB
}

// NewEmployeeRepository creates an EmployeeRepository backed by db.
func NewEmployeeRepository(db *sql.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

const employeeColumns = `id, employee_no, name, department, is_active, created_at, updated_at, deleted_at`

func scanEmployee(row *sql.Row, e *model.Employee) error {
	return row.Scan(&e.ID, &e.EmployeeNo, &e.Name, &e.Department, &e.IsActive,
		&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt)
}

// Create inserts a new employee using tx.
func (r *EmployeeRepository) Create(ctx context.Context, tx DBTX, e *model.Employee) error {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO employees (employee_no, name, department, is_active) VALUES (?, ?, ?, ?)`,
		e.EmployeeNo, e.Name, e.Department, e.IsActive)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = id
	return nil
}

// FindByID returns a non-deleted employee by id, or ErrNotFound.
func (r *EmployeeRepository) FindByID(ctx context.Context, q DBTX, id int64) (*model.Employee, error) {
	var e model.Employee
	err := scanEmployee(q.QueryRowContext(ctx,
		`SELECT `+employeeColumns+` FROM employees WHERE id = ? AND deleted_at IS NULL`, id), &e)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}

// FindByEmployeeNo returns a non-deleted employee by employee_no, or ErrNotFound.
func (r *EmployeeRepository) FindByEmployeeNo(ctx context.Context, q DBTX, employeeNo string) (*model.Employee, error) {
	var e model.Employee
	err := scanEmployee(q.QueryRowContext(ctx,
		`SELECT `+employeeColumns+` FROM employees WHERE employee_no = ? AND deleted_at IS NULL`, employeeNo), &e)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}

// List returns employees ordered by id, optionally filtered to active only.
func (r *EmployeeRepository) List(ctx context.Context, q DBTX, activeOnly bool, limit, offset int) ([]model.Employee, error) {
	query := `SELECT ` + employeeColumns + ` FROM employees WHERE deleted_at IS NULL`
	if activeOnly {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY id ASC LIMIT ? OFFSET ?`
	rows, err := q.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Employee{}
	for rows.Next() {
		var e model.Employee
		if err := rows.Scan(&e.ID, &e.EmployeeNo, &e.Name, &e.Department, &e.IsActive,
			&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Update modifies name, department and active flag using tx.
func (r *EmployeeRepository) Update(ctx context.Context, tx DBTX, e *model.Employee) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE employees SET name = ?, department = ?, is_active = ? WHERE id = ? AND deleted_at IS NULL`,
		e.Name, e.Department, e.IsActive, e.ID)
	return err
}

// SoftDelete marks an employee as deleted using tx.
func (r *EmployeeRepository) SoftDelete(ctx context.Context, tx DBTX, id int64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE employees SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL`, id)
	return err
}
