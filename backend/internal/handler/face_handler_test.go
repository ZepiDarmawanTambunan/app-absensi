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
	err        error
}

func (f *fakeFaceService) Enroll(_ context.Context, _ int64, _ [][]float32, _ []float64) (*model.FaceEnrollment, error) {
	return f.enrollment, f.err
}

func (f *fakeFaceService) Verify(_ context.Context, _ int64, _ []float32, _ *float64) (*service.VerifyResult, error) {
	return f.result, f.err
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
			"employee_id": 1,
			"embedding":   testEmbedding(128),
		})
	}

	t.Run("success returns 200 with match", func(t *testing.T) {
		h := NewFaceHandler(&fakeFaceService{result: &service.VerifyResult{Match: true, Distance: 0.12, Threshold: 0.5}})
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
