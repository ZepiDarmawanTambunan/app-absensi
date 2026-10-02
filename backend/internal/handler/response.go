// Package handler implements the HTTP handlers. Handlers are thin:
// parse and validate input, call the service, render the JSON envelope.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/auth"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/service"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: data})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Error: &apiError{Code: code, Message: message}})
}

// decodeJSON parses the request body (max 1 MB). It returns false and
// writes a 400 response when the body is not valid JSON.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return false
	}
	return true
}

// statusFor maps domain errors to HTTP status codes.
func statusFor(err error) int {
	var ve service.ValidationError
	var locked *service.FaceLockedError
	switch {
	case errors.As(err, &ve):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrChallengeNotFound),
		errors.Is(err, service.ErrChallengeExpired),
		errors.Is(err, service.ErrChallengeUsed):
		return http.StatusBadRequest
	case errors.As(err, &locked):
		return http.StatusLocked
	case errors.Is(err, repository.ErrNotFound),
		errors.Is(err, service.ErrEmployeeNotFound),
		errors.Is(err, service.ErrNoOpenCheckIn),
		errors.Is(err, service.ErrNoFaceEnrollment):
		return http.StatusNotFound
	case errors.Is(err, service.ErrInvalidCredentials),
		errors.Is(err, service.ErrInvalidToken),
		errors.Is(err, auth.ErrInvalidToken):
		return http.StatusUnauthorized
	case errors.Is(err, service.ErrEmailTaken),
		errors.Is(err, service.ErrEmployeeNoTaken),
		errors.Is(err, service.ErrAlreadyCheckedIn):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func errorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusLocked:
		return "locked"
	default:
		return "internal_error"
	}
}

// writeServiceError renders a service-layer error as a JSON error response.
// Internal errors are masked so implementation details never leak.
func writeServiceError(w http.ResponseWriter, err error) {
	status := statusFor(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "internal server error"
	}
	writeError(w, status, errorCode(status), msg)
}
