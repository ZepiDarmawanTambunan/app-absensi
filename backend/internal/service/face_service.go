package service

import (
	"context"
	"errors"
	"time"

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

// FaceService manages face enrollment, 1:1 verification, and the
// Milestone 3 anti-spoofing pipeline (liveness enforcement + lockout).
// Verification is always 1:1 against the claimed employee's template —
// never 1:N against the whole employee database.
type FaceService struct {
	db             repository.DBTX
	employees      EmployeeStore
	faces          FaceStore
	spoofs         SpoofStore
	transact       Transactor
	matchThreshold float64
	liveness       LivenessOptions
	now            func() time.Time
}

// NewFaceService creates a FaceService. Out-of-range thresholds and
// non-positive lockout settings fall back to the documented defaults.
func NewFaceService(db repository.DBTX, employees EmployeeStore, faces FaceStore, spoofs SpoofStore, transact Transactor, matchThreshold float64, liveness LivenessOptions) *FaceService {
	if matchThreshold <= 0 || matchThreshold >= 2 {
		matchThreshold = DefaultMatchThreshold
	}
	if liveness.Threshold <= 0 || liveness.Threshold > 1 {
		liveness.Threshold = DefaultLivenessThreshold
	}
	if liveness.MaxFailures <= 0 {
		liveness.MaxFailures = DefaultMaxLivenessFailures
	}
	if liveness.LockoutMinutes <= 0 {
		liveness.LockoutMinutes = DefaultLivenessLockoutMinutes
	}
	if liveness.Challenges == nil {
		liveness.Challenges = NewChallengeStore(ChallengeTTL)
	}
	return &FaceService{
		db: db, employees: employees, faces: faces, spoofs: spoofs,
		transact: transact, matchThreshold: matchThreshold,
		liveness: liveness, now: time.Now,
	}
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

// VerifyInput bundles one face-verification attempt.
type VerifyInput struct {
	EmployeeID    int64
	Embedding     []float32
	LivenessScore *float64
	DeviceID      string
	ChallengeID   string
}

// LivenessInfo describes the liveness gate outcome of a verification.
type LivenessInfo struct {
	Score     float64 `json:"score"`
	Threshold float64 `json:"threshold"`
	Passed    bool    `json:"passed"`
}

// VerifyResult is the outcome of a 1:1 face verification.
type VerifyResult struct {
	Match        bool          `json:"match"`
	Distance     float64       `json:"distance"`
	Threshold    float64       `json:"threshold"`
	Liveness     *LivenessInfo `json:"liveness,omitempty"`
	RejectReason string        `json:"reject_reason,omitempty"`
}

// Verify compares the probe embedding against the claimed employee's
// stored template, enforcing the Milestone 3 anti-spoofing policy first:
// a missing/invalid liveness score is rejected, a score below threshold
// is logged to spoof_attempts and rejected without comparing the
// embedding, and repeated failures temporarily lock verification
// (FaceLockedError → HTTP 423).
func (s *FaceService) Verify(ctx context.Context, in VerifyInput) (*VerifyResult, error) {
	if in.EmployeeID <= 0 {
		return nil, ValidationError{Field: "employee_id", Message: "employee_id is required"}
	}
	if len(in.Embedding) == 0 {
		return nil, ValidationError{Field: "embedding", Message: "embedding is required"}
	}
	emp, err := s.employees.FindByID(ctx, s.db, in.EmployeeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEmployeeNotFound
	}
	if err != nil {
		return nil, err
	}
	if !emp.IsActive {
		return nil, ErrEmployeeInactive
	}

	// Lockout is checked before any liveness or embedding work, so a
	// locked account cannot burn challenges or probe the matcher.
	if locked, retryAfter, err := s.checkLockout(ctx, in.EmployeeID); err != nil {
		return nil, err
	} else if locked {
		return nil, &FaceLockedError{RetryAfter: retryAfter}
	}

	// A supplied challenge must be valid; it is consumed single-use to
	// prevent replay of one liveness performance across many verifies.
	if in.ChallengeID != "" {
		if err := s.liveness.Challenges.Consume(in.ChallengeID); err != nil {
			return nil, err
		}
	}

	live, err := s.gateLiveness(ctx, in)
	if err != nil {
		return nil, err
	}
	if live != nil && !live.Passed {
		return &VerifyResult{
			Match:        false,
			Threshold:    s.matchThreshold,
			Liveness:     live,
			RejectReason: LivenessRejectReason,
		}, nil
	}

	tpl, err := s.faces.FindByEmployeeID(ctx, s.db, in.EmployeeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNoFaceEnrollment
	}
	if err != nil {
		return nil, err
	}
	dist, err := face.CosineDistance(in.Embedding, tpl.Embedding)
	if err != nil {
		return nil, ValidationError{Field: "embedding", Message: err.Error()}
	}
	return &VerifyResult{
		Match:     dist < s.matchThreshold,
		Distance:  dist,
		Threshold: s.matchThreshold,
		Liveness:  live,
	}, nil
}

// gateLiveness enforces the liveness policy. It returns (nil, nil) when
// liveness is not required and no score was supplied. A failed gate is
// recorded in the spoof_attempts audit log before returning.
func (s *FaceService) gateLiveness(ctx context.Context, in VerifyInput) (*LivenessInfo, error) {
	if !s.liveness.Required && in.LivenessScore == nil {
		return nil, nil
	}
	if in.LivenessScore == nil {
		return nil, ValidationError{Field: "liveness_score", Message: "liveness_score is required"}
	}
	score := *in.LivenessScore
	if score < 0 || score > 1 {
		return nil, ValidationError{Field: "liveness_score", Message: "liveness_score must be between 0 and 1"}
	}
	info := &LivenessInfo{Score: score, Threshold: s.liveness.Threshold, Passed: score >= s.liveness.Threshold}
	if info.Passed {
		return info, nil
	}
	var deviceID *string
	if in.DeviceID != "" {
		d := in.DeviceID
		deviceID = &d
	}
	err := s.transact(ctx, func(tx repository.DBTX) error {
		return s.spoofs.Record(ctx, tx, &model.SpoofAttempt{
			EmployeeID:    in.EmployeeID,
			LivenessScore: score,
			Reason:        LivenessRejectReason,
			DeviceID:      deviceID,
		})
	})
	if err != nil {
		return nil, err
	}
	return info, nil
}

// checkLockout reports whether face verification is currently locked for
// the employee, with the remaining wait. The lock engages once
// MaxFailures rejections accumulate inside LivenessFailureWindow and
// lifts LockoutMinutes after the most recent rejection — no manual
// reset is needed.
func (s *FaceService) checkLockout(ctx context.Context, employeeID int64) (bool, time.Duration, error) {
	now := s.now()
	n, lastAt, err := s.spoofs.CountRecent(ctx, s.db, employeeID, now.Add(-LivenessFailureWindow))
	if err != nil {
		return false, 0, err
	}
	if n < s.liveness.MaxFailures || lastAt == nil {
		return false, 0, nil
	}
	lockDur := time.Duration(s.liveness.LockoutMinutes) * time.Minute
	if elapsed := now.Sub(*lastAt); elapsed >= lockDur {
		return false, 0, nil
	} else {
		return true, lockDur - elapsed, nil
	}
}

// IsFaceLocked reports whether face verification is currently locked for
// the employee, and how long until the lock lifts.
func (s *FaceService) IsFaceLocked(ctx context.Context, employeeID int64) (bool, time.Duration, error) {
	return s.checkLockout(ctx, employeeID)
}

// IssueChallenge issues a single-use liveness challenge for the mobile app.
func (s *FaceService) IssueChallenge() *Challenge {
	return s.liveness.Challenges.Issue()
}
