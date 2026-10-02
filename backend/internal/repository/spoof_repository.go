package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
)

// SpoofAttemptRepository persists the anti-spoofing audit log.
type SpoofAttemptRepository struct {
	db *sql.DB
}

// NewSpoofAttemptRepository creates a SpoofAttemptRepository backed by db.
func NewSpoofAttemptRepository(db *sql.DB) *SpoofAttemptRepository {
	return &SpoofAttemptRepository{db: db}
}

// Record inserts one rejected liveness check using tx.
func (r *SpoofAttemptRepository) Record(ctx context.Context, tx DBTX, a *model.SpoofAttempt) error {
	var deviceID any
	if a.DeviceID != nil {
		deviceID = *a.DeviceID
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO spoof_attempts (employee_id, liveness_score, reason, device_id) VALUES (?, ?, ?, ?)`,
		a.EmployeeID, a.LivenessScore, a.Reason, deviceID)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = id
	return nil
}

// CountRecent returns how many spoof attempts the employee has at or after
// `since`, plus the timestamp of the most recent one (nil when none).
func (r *SpoofAttemptRepository) CountRecent(ctx context.Context, q DBTX, employeeID int64, since time.Time) (int, *time.Time, error) {
	var n int
	var last sql.NullTime
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*), MAX(created_at) FROM spoof_attempts WHERE employee_id = ? AND created_at >= ?`,
		employeeID, since).Scan(&n, &last)
	if err != nil {
		return 0, nil, err
	}
	if !last.Valid {
		return n, nil, nil
	}
	t := last.Time
	return n, &t, nil
}
