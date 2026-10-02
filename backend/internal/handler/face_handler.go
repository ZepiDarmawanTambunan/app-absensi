package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/service"
)

// FaceService is the contract FaceHandler needs.
type FaceService interface {
	Enroll(ctx context.Context, employeeID int64, embeddings [][]float32, qualityScores []float64) (*model.FaceEnrollment, error)
	Verify(ctx context.Context, in service.VerifyInput) (*service.VerifyResult, error)
	IssueChallenge() *service.Challenge
}

// FaceHandler serves the /face endpoints.
type FaceHandler struct {
	svc FaceService
}

// NewFaceHandler creates a FaceHandler.
func NewFaceHandler(svc FaceService) *FaceHandler {
	return &FaceHandler{svc: svc}
}

type enrollFaceRequest struct {
	EmployeeID    int64       `json:"employee_id"`
	Embeddings    [][]float32 `json:"embeddings"`
	QualityScores []float64   `json:"quality_scores"`
}

type verifyFaceRequest struct {
	EmployeeID    int64     `json:"employee_id"`
	Embedding     []float32 `json:"embedding"`
	LivenessScore *float64  `json:"liveness_score"`
	DeviceID      string    `json:"device_id,omitempty"`
	ChallengeID   string    `json:"challenge_id,omitempty"`
}

// Enroll handles POST /api/v1/face/enroll.
func (h *FaceHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	var req enrollFaceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	e, err := h.svc.Enroll(r.Context(), req.EmployeeID, req.Embeddings, req.QualityScores)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// Verify handles POST /api/v1/face/verify.
// A locked account (FaceLockedError) yields HTTP 423 with the
// remaining wait time; every other service error uses the standard mapping.
func (h *FaceHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req verifyFaceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := h.svc.Verify(r.Context(), service.VerifyInput{
		EmployeeID:    req.EmployeeID,
		Embedding:     req.Embedding,
		LivenessScore: req.LivenessScore,
		DeviceID:      req.DeviceID,
		ChallengeID:   req.ChallengeID,
	})
	if err != nil {
		var locked *service.FaceLockedError
		if errors.As(err, &locked) {
			secs := int64(locked.RetryAfter.Round(time.Second).Seconds())
			if secs < 1 {
				secs = 1
			}
			writeJSON(w, http.StatusLocked, map[string]any{
				"locked":      true,
				"retry_after": secs,
				"message":     locked.Error(),
			})
			return
		}
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// Challenge handles GET /api/v1/face/challenge. It issues a single-use
// liveness challenge (blink / turn_head) for the mobile app to perform.
// The challenge id is sent back with the verify call.
func (h *FaceHandler) Challenge(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.IssueChallenge())
}
