package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
)

// AttendanceRepository persists attendance logs and devices.
type AttendanceRepository struct {
	db *sql.DB
}

// NewAttendanceRepository creates an AttendanceRepository backed by db.
func NewAttendanceRepository(db *sql.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

const attendanceColumns = `id, employee_id, type, method, verified, liveness_score, recorded_at`

func scanAttendanceLog(row *sql.Row, l *model.AttendanceLog) error {
	return row.Scan(&l.ID, &l.EmployeeID, &l.Type, &l.Method, &l.Verified, &l.LivenessScore, &l.RecordedAt)
}

// CreateLog inserts an attendance log using tx.
func (r *AttendanceRepository) CreateLog(ctx context.Context, tx DBTX, l *model.AttendanceLog) error {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO attendance_logs (employee_id, type, method, verified, liveness_score, recorded_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		l.EmployeeID, l.Type, l.Method, l.Verified, l.LivenessScore, l.RecordedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	l.ID = id
	return nil
}

// FindOpenCheckIn returns the latest check-in that has no later check-out,
// or ErrNotFound when the employee is not currently checked in.
func (r *AttendanceRepository) FindOpenCheckIn(ctx context.Context, q DBTX, employeeID int64) (*model.AttendanceLog, error) {
	var l model.AttendanceLog
	err := scanAttendanceLog(q.QueryRowContext(ctx,
		`SELECT `+attendanceColumns+`
		 FROM attendance_logs AS ci
		 WHERE ci.employee_id = ? AND ci.type = 'check_in'
		   AND NOT EXISTS (
		       SELECT 1 FROM attendance_logs AS co
		       WHERE co.employee_id = ci.employee_id
		         AND co.type = 'check_out'
		         AND co.recorded_at >= ci.recorded_at
		   )
		 ORDER BY ci.recorded_at DESC
		 LIMIT 1`, employeeID), &l)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &l, err
}

// ListByEmployee returns logs for an employee in a time range, newest first.
func (r *AttendanceRepository) ListByEmployee(ctx context.Context, q DBTX, employeeID int64, from, to time.Time, limit, offset int) ([]model.AttendanceLog, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT `+attendanceColumns+`
		 FROM attendance_logs
		 WHERE employee_id = ? AND recorded_at >= ? AND recorded_at < ?
		 ORDER BY recorded_at DESC
		 LIMIT ? OFFSET ?`, employeeID, from, to, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.AttendanceLog{}
	for rows.Next() {
		var l model.AttendanceLog
		if err := rows.Scan(&l.ID, &l.EmployeeID, &l.Type, &l.Method, &l.Verified, &l.LivenessScore, &l.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetEmployeeByNo returns a non-deleted, active-or-not employee by employee_no.
func (r *AttendanceRepository) GetEmployeeByNo(ctx context.Context, q DBTX, employeeNo string) (*model.Employee, error) {
	var e model.Employee
	err := q.QueryRowContext(ctx,
		`SELECT id, employee_no, name, department, is_active, created_at, updated_at, deleted_at
		 FROM employees WHERE employee_no = ? AND deleted_at IS NULL`, employeeNo).
		Scan(&e.ID, &e.EmployeeNo, &e.Name, &e.Department, &e.IsActive, &e.CreatedAt, &e.UpdatedAt, &e.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}

// UpsertDevice records a device sighting using tx.
func (r *AttendanceRepository) UpsertDevice(ctx context.Context, tx DBTX, d *model.Device) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO devices (device_id, employee_id, platform, last_seen)
		 VALUES (?, ?, ?, NOW())
		 ON DUPLICATE KEY UPDATE employee_id = VALUES(employee_id), platform = VALUES(platform), last_seen = NOW()`,
		d.DeviceID, d.EmployeeID, d.Platform)
	return err
}
