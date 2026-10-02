package service

import (
	"context"
	"errors"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/face"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

var (
	// ErrNoFaceEnrollment is returned when the employee has no face template.
	ErrNoFaceEnrollment = errors.New("service: employee has no face enrollment")
)

// DefaultMatchThreshold is the cosine-distance acceptance threshold.
// Calibrate it on a labeled set and report FAR/FRR before changing it —
// never lower it just to "fix" verification failures.
const DefaultMatchThreshold = 0.50

// MinEnrollQuality is the minimum per-frame quality score accepted during
// enrollment. Frames below it are discarded; at least one frame must pass.
const MinEnrollQuality = 0.5

// FaceStore is the persistence contract FaceService needs.
type FaceStore interface {
	Replace(ctx context.Context, tx repository.DBTX, e *model.FaceEnrollment) error
	FindByEmployeeID(ctx context.Context, q repository.DBTX, employeeID int64) (*model.FaceEnrollment, error)
}

// FaceService manages face enrollment and 1:1 verification.
// Verification is always 1:1 against the claimed employee's template —
// never 1:N against the whole employee database.
type FaceService struct {
	db             repository.DBTX
	employees      EmployeeStore
	faces          FaceStore
	transact       Transactor
	matchThreshold float64
}

// NewFaceService creates a FaceService.
func NewFaceService(db repository.DBTX, employees EmployeeStore, faces FaceStore, transact Transactor, matchThreshold float64) *FaceService {
	if matchThreshold <= 0 || matchThreshold >= 2 {
		matchThreshold = DefaultMatchThreshold
	}
	return &FaceService{db: db, employees: employees, faces: faces, transact: transact, matchThreshold: matchThreshold}
}

// Enroll validates the captured embeddings, averages the frames that pass
// the quality bar into one L2-normalized template, and stores it,
// replacing any previous template of the employee in one transaction.
func (s *FaceService) Enroll(ctx context.Context, employeeID int64, embeddings [][]float32, qualityScores []float64) (*model.FaceEnrollment, error) {
	if employeeID <= 0 {
		return nil, ValidationError{Field: "employee_id", Message: "employee_id is required"}
	}
	if len(embeddings) == 0 {
		return nil, ValidationError{Field: "embeddings", Message: "at least one embedding is required"}
	}
	if len(qualityScores) != len(embeddings) {
		return nil, ValidationError{Field: "quality_scores", Message: "quality_scores must align with embeddings"}
	}
	dim := len(embeddings[0])
	if !face.IsSupportedDimension(dim) {
		return nil, ValidationError{Field: "embeddings", Message: "unsupported embedding dimension"}
	}
	var good [][]float32
	var qualitySum float64
	for i, e := range embeddings {
		if len(e) != dim {
			return nil, ValidationError{Field: "embeddings", Message: "all embeddings must share one dimension"}
		}
		if qualityScores[i] >= MinEnrollQuality {
			good = append(good, e)
			qualitySum += qualityScores[i]
		}
	}
	if len(good) == 0 {
		return nil, ValidationError{Field: "quality_scores", Message: "no frame passed the quality bar"}
	}
	template, err := face.Average(good)
	if err != nil {
		return nil, err
	}
	var enrolled *model.FaceEnrollment
	err = s.transact(ctx, func(tx repository.DBTX) error {
		emp, err := s.employees.FindByID(ctx, tx, employeeID)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrEmployeeNotFound
		}
		if err != nil {
			return err
		}
		if !emp.IsActive {
			return ErrEmployeeInactive
		}
		enrolled = &model.FaceEnrollment{
			EmployeeID:   employeeID,
			Embedding:    template,
			Dimension:    dim,
			QualityScore: qualitySum / float64(len(good)),
		}
		return s.faces.Replace(ctx, tx, enrolled)
	})
	if err != nil {
		return nil, err
	}
	return enrolled, nil
}

// VerifyResult is the outcome of a 1:1 face verification.
type VerifyResult struct {
	Match     bool    `json:"match"`
	Distance  float64 `json:"distance"`
	Threshold float64 `json:"threshold"`
}

// Verify compares the probe embedding against the claimed employee's
// stored template. livenessScore is accepted for the Milestone 3
// anti-spoofing pipeline; it is validated here but not yet enforced.
func (s *FaceService) Verify(ctx context.Context, employeeID int64, embedding []float32, livenessScore *float64) (*VerifyResult, error) {
	if employeeID <= 0 {
		return nil, ValidationError{Field: "employee_id", Message: "employee_id is required"}
	}
	if len(embedding) == 0 {
		return nil, ValidationError{Field: "embedding", Message: "embedding is required"}
	}
	if livenessScore != nil && (*livenessScore < 0 || *livenessScore > 1) {
		return nil, ValidationError{Field: "liveness_score", Message: "liveness_score must be between 0 and 1"}
	}
	emp, err := s.employees.FindByID(ctx, s.db, employeeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEmployeeNotFound
	}
	if err != nil {
		return nil, err
	}
	if !emp.IsActive {
		return nil, ErrEmployeeInactive
	}
	tpl, err := s.faces.FindByEmployeeID(ctx, s.db, employeeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNoFaceEnrollment
	}
	if err != nil {
		return nil, err
	}
	dist, err := face.CosineDistance(embedding, tpl.Embedding)
	if err != nil {
		return nil, ValidationError{Field: "embedding", Message: err.Error()}
	}
	return &VerifyResult{
		Match:     dist < s.matchThreshold,
		Distance:  dist,
		Threshold: s.matchThreshold,
	}, nil
}
