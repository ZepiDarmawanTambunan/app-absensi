package handler

import (
	"context"
	"net/http"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/service"
)

// FaceService is the contract FaceHandler needs.
type FaceService interface {
	Enroll(ctx context.Context, employeeID int64, embeddings [][]float32, qualityScores []float64) (*model.FaceEnrollment, error)
	Verify(ctx context.Context, employeeID int64, embedding []float32, livenessScore *float64) (*service.VerifyResult, error)
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
func (h *FaceHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req verifyFaceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := h.svc.Verify(r.Context(), req.EmployeeID, req.Embedding, req.LivenessScore)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
