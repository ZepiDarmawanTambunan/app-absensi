// Package middleware provides HTTP middleware: request IDs, structured
// logging, panic recovery and JWT authentication.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/auth"
)

type ctxKey string

const (
	// RequestIDKey is the context key for the request ID.
	RequestIDKey ctxKey = "request_id"
	// UserKey is the context key for the authenticated *auth.Claims.
	UserKey ctxKey = "user"
)

// RequestID assigns an ID to every request (reusing X-Request-ID when set).
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), RequestIDKey, id)))
	})
}

// Logger emits one structured log line per request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"request_id", r.Context().Value(RequestIDKey),
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

// Recoverer converts panics into 500 JSON responses.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "request_id", r.Context().Value(RequestIDKey), "panic", rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{"code": "internal_error", "message": "internal server error"},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Auth validates the Bearer JWT and injects *auth.Claims into the context.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
				unauthorized(w, "missing or malformed Authorization header")
				return
			}
			claims, err := auth.ParseToken(strings.TrimSpace(parts[1]), jwtSecret)
			if err != nil || claims.TokenType != auth.TokenTypeAccess {
				unauthorized(w, "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserKey, claims)))
		})
	}
}

// ClaimsFromContext returns the authenticated claims, or nil.
func ClaimsFromContext(ctx context.Context) *auth.Claims {
	if c, ok := ctx.Value(UserKey).(*auth.Claims); ok {
		return c
	}
	return nil
}

func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": "unauthorized", "message": message},
	})
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
