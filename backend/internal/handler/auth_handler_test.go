package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/service"
)

// fakeAuthService implements handler.AuthService for tests.
type fakeAuthService struct {
	tokens *model.TokenPair
	err    error
}

func (f *fakeAuthService) Register(_ context.Context, _, _, _ string) (*model.TokenPair, error) {
	return f.tokens, f.err
}

func (f *fakeAuthService) Login(_ context.Context, _, _ string) (*model.TokenPair, error) {
	return f.tokens, f.err
}

func (f *fakeAuthService) Refresh(_ context.Context, _ string) (*model.TokenPair, error) {
	return f.tokens, f.err
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) (map[string]any, map[string]any) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	data, _ := body["data"].(map[string]any)
	errObj, _ := body["error"].(map[string]any)
	return data, errObj
}

func TestAuthHandler_Register(t *testing.T) {
	tokens := &model.TokenPair{AccessToken: "acc", RefreshToken: "ref", ExpiresIn: 900}
	h := NewAuthHandler(&fakeAuthService{tokens: tokens})

	t.Run("success returns 201 with token pair", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
			bytes.NewBufferString(`{"name":"Admin","email":"admin@example.com","password":"secret123"}`))
		rec := httptest.NewRecorder()
		h.Register(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}
		data, errObj := decodeEnvelope(t, rec)
		if errObj != nil {
			t.Fatalf("expected no error object, got %v", errObj)
		}
		if data["access_token"] != "acc" || data["refresh_token"] != "ref" {
			t.Fatalf("unexpected data: %v", data)
		}
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
			bytes.NewBufferString(`{not json`))
		rec := httptest.NewRecorder()
		h.Register(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		_, errObj := decodeEnvelope(t, rec)
		if errObj == nil || errObj["code"] != "bad_request" {
			t.Fatalf("expected bad_request error object, got %v", errObj)
		}
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		dup := NewAuthHandler(&fakeAuthService{err: service.ErrEmailTaken})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
			bytes.NewBufferString(`{"name":"Admin","email":"a@b.c","password":"secret123"}`))
		rec := httptest.NewRecorder()
		dup.Register(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", rec.Code)
		}
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		bad := NewAuthHandler(&fakeAuthService{err: service.ValidationError{Field: "email", Message: "email is not valid"}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
			bytes.NewBufferString(`{"name":"Admin","email":"x","password":"secret123"}`))
		rec := httptest.NewRecorder()
		bad.Register(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestAuthHandler_Login(t *testing.T) {
	tokens := &model.TokenPair{AccessToken: "acc", RefreshToken: "ref", ExpiresIn: 900}

	t.Run("bad credentials return 401", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{err: service.ErrInvalidCredentials})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
			bytes.NewBufferString(`{"email":"a@b.c","password":"wrong"}`))
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		_, errObj := decodeEnvelope(t, rec)
		if errObj == nil || errObj["code"] != "unauthorized" {
			t.Fatalf("expected unauthorized error object, got %v", errObj)
		}
	})

	t.Run("success returns 200", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{tokens: tokens})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
			bytes.NewBufferString(`{"email":"a@b.c","password":"secret123"}`))
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	tokens := &model.TokenPair{AccessToken: "acc2", RefreshToken: "ref2", ExpiresIn: 900}

	t.Run("missing token returns 400", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{tokens: tokens})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
			bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()
		h.Refresh(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("invalid refresh token returns 401", func(t *testing.T) {
		h := NewAuthHandler(&fakeAuthService{err: service.ErrInvalidToken})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh",
			bytes.NewBufferString(`{"refresh_token":"bad"}`))
		rec := httptest.NewRecorder()
		h.Refresh(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})
}
