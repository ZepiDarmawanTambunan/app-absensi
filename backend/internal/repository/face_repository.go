package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/face"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
)

// FaceEnrollmentRepository persists face templates.
type FaceEnrollmentRepository struct {
	db *sql.DB
}

// NewFaceEnrollmentRepository creates a FaceEnrollmentRepository backed by db.
func NewFaceEnrollmentRepository(db *sql.DB) *FaceEnrollmentRepository {
	return &FaceEnrollmentRepository{db: db}
}

const faceEnrollmentColumns = `id, employee_id, embedding, dimension, quality_score, enrolled_at`

func scanFaceEnrollment(row *sql.Row, e *model.FaceEnrollment) error {
	var blob []byte
	if err := row.Scan(&e.ID, &e.EmployeeID, &blob, &e.Dimension, &e.QualityScore, &e.EnrolledAt); err != nil {
		return err
	}
	emb, err := face.DecodeEmbedding(blob)
	if err != nil {
		return err
	}
	e.Embedding = emb
	return nil
}

// Replace soft-deletes the employee's existing templates and stores the
// new one, atomically, using tx.
func (r *FaceEnrollmentRepository) Replace(ctx context.Context, tx DBTX, e *model.FaceEnrollment) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE face_enrollments SET deleted_at = NOW() WHERE employee_id = ? AND deleted_at IS NULL`,
		e.EmployeeID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO face_enrollments (employee_id, embedding, dimension, quality_score) VALUES (?, ?, ?, ?)`,
		e.EmployeeID, face.EncodeEmbedding(e.Embedding), e.Dimension, e.QualityScore)
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

// FindByEmployeeID returns the employee's latest active template, or ErrNotFound.
func (r *FaceEnrollmentRepository) FindByEmployeeID(ctx context.Context, q DBTX, employeeID int64) (*model.FaceEnrollment, error) {
	var e model.FaceEnrollment
	err := scanFaceEnrollment(q.QueryRowContext(ctx,
		`SELECT `+faceEnrollmentColumns+` FROM face_enrollments WHERE employee_id = ? AND deleted_at IS NULL ORDER BY id DESC LIMIT 1`,
		employeeID), &e)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}
