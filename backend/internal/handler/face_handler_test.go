package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/middleware"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/service"
)

// fakeFaceService implements handler.FaceService for tests.
type fakeFaceService struct {
	enrollment *model.FaceEnrollment
	result     *service.VerifyResult
	challenge  *service.Challenge
	err        error
}

func (f *fakeFaceService) Enroll(_ context.Context, _ int64, _ [][]float32, _ []float64) (*model.FaceEnrollment, error) {
	return f.enrollment, f.err
}

func (f *fakeFaceService) Verify(_ context.Context, _ service.VerifyInput) (*service.VerifyResult, error) {
	return f.result, f.err
}

func (f *fakeFaceService) IssueChallenge() *service.Challenge {
	return f.challenge
}

func testEmbedding(dim int) []float32 {
	e := make([]float32, dim)
	for i := range e {
		e[i] = 0.01
	}
	return e
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func TestFaceHandler_Enroll(t *testing.T) {
	body := func() *bytes.Buffer {
		return jsonBody(t, map[string]any{
			"employee_id":    1,
			"embeddings":     [][]float32{testEmbedding(128), testEmbedding(128)},
			"quality_scores": []float64{0.9, 0.8},
		})
	}

	t.Run("success returns 201", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{enrollment: &model.FaceEnrollment{
			ID: 1, EmployeeID: 1, Dimension: 128, QualityScore: 0.85, EnrolledAt: time.Now(),
		}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/enroll", body())
		rec := httptest.NewRecorder()
		h.Enroll(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}
		data, errObj := decodeEnvelope(t, rec)
		if errObj != nil {
			t.Fatalf("expected no error, got %v", errObj)
		}
		if data["dimension"] != float64(128) {
			t.Fatalf("unexpected data: %v", data)
		}
		if _, present := data["Embedding"]; present {
			t.Fatal("embedding must never be exposed in JSON")
		}
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{err: service.ValidationError{Field: "embeddings", Message: "bad"}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/enroll", body())
		rec := httptest.NewRecorder()
		h.Enroll(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("unknown employee returns 404", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{err: service.ErrEmployeeNotFound})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/enroll", body())
		rec := httptest.NewRecorder()
		h.Enroll(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/enroll",
			bytes.NewBufferString(`{"employee_id":`))
		rec := httptest.NewRecorder()
		h.Enroll(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestFaceHandler_Verify(t *testing.T) {
	body := func() *bytes.Buffer {
		return jsonBody(t, map[string]any{
			"employee_id":    1,
			"embedding":      testEmbedding(128),
			"liveness_score": 0.92,
			"device_id":      "dev-1",
		})
	}

	t.Run("success returns 200 with match and liveness", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{result: &service.VerifyResult{
			Match: true, Distance: 0.12, Threshold: 0.5,
			Liveness: &service.LivenessInfo{Score: 0.92, Threshold: 0.7, Passed: true},
		}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/verify", body())
		rec := httptest.NewRecorder()
		h.Verify(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, _ := decodeEnvelope(t, rec)
		if data["match"] != true {
			t.Fatalf("expected match=true, got %v", data)
		}
		live, ok := data["liveness"].(map[string]any)
		if !ok || live["passed"] != true {
			t.Fatalf("expected passed liveness info, got %v", data["liveness"])
		}
	})

	t.Run("liveness rejection returns 200 with match=false", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{result: &service.VerifyResult{
			Match: false, Threshold: 0.5, RejectReason: service.LivenessRejectReason,
			Liveness: &service.LivenessInfo{Score: 0.3, Threshold: 0.7, Passed: false},
		}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/verify", body())
		rec := httptest.NewRecorder()
		h.Verify(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, _ := decodeEnvelope(t, rec)
		if data["match"] != false || data["reject_reason"] != service.LivenessRejectReason {
			t.Fatalf("unexpected data: %v", data)
		}
	})

	t.Run("locked account returns 423 with retry_after", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{err: &service.FaceLockedError{RetryAfter: 14 * time.Minute}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/verify", body())
		rec := httptest.NewRecorder()
		h.Verify(rec, req)
		if rec.Code != http.StatusLocked {
			t.Fatalf("expected 423, got %d", rec.Code)
		}
		data, errObj := decodeEnvelope(t, rec)
		if errObj != nil {
			t.Fatalf("expected data payload, got error %v", errObj)
		}
		if data["locked"] != true {
			t.Fatalf("expected locked=true, got %v", data)
		}
		if data["retry_after"] != float64(840) {
			t.Fatalf("expected retry_after 840, got %v", data["retry_after"])
		}
	})

	t.Run("missing liveness score returns 400", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{err: service.ValidationError{Field: "liveness_score", Message: "liveness_score is required"}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/verify", body())
		rec := httptest.NewRecorder()
		h.Verify(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("no enrollment returns 404", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{err: service.ErrNoFaceEnrollment})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/verify", body())
		rec := httptest.NewRecorder()
		h.Verify(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/verify",
			bytes.NewBufferString(`{"employee_id":`))
		rec := httptest.NewRecorder()
		h.Verify(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestFaceHandler_Challenge(t *testing.T) {
	h := NewFaceHandler(&fakeFaceService{challenge: &service.Challenge{
		ID: "ch-123", Type: service.ChallengeBlink, ExpiresIn: 120,
	}})

	t.Run("returns 200 with challenge payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/face/challenge", nil)
		rec := httptest.NewRecorder()
		h.Challenge(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, errObj := decodeEnvelope(t, rec)
		if errObj != nil {
			t.Fatalf("expected no error, got %v", errObj)
		}
		if data["challenge_id"] != "ch-123" || data["type"] != "blink" || data["expires_in"] != float64(120) {
			t.Fatalf("unexpected data: %v", data)
		}
	})
}

func TestFaceHandler_RequiresAuth(t *testing.T) {
	h := NewFaceHandler(&fakeFaceService{})
	protected := middleware.Auth("test-secret")(http.HandlerFunc(h.Verify))

	t.Run("missing token returns 401", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"employee_id": 1, "embedding": testEmbedding(128)})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/verify", body)
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("garbage token returns 401", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"employee_id": 1, "embedding": testEmbedding(128)})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/face/verify", body)
		req.Header.Set("Authorization", "Bearer not-a-token")
		rec := httptest.NewRecorder()
		protected.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})
}
