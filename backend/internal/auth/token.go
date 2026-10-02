// Package auth owns JWT issuance and validation shared by the
// service layer and the HTTP middleware.
package auth

import (
	"errors"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

// Token types.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// ErrInvalidToken is returned when a token cannot be validated.
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims are the JWT claims used by this service.
type Claims struct {
	jwt.RegisteredClaims
	UserID    int64  `json:"uid"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"typ"`
}

// IssueToken creates a signed JWT for the given user.
func IssueToken(u *model.User, secret, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.Email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID:    u.ID,
		Email:     u.Email,
		Role:      u.Role,
		TokenType: tokenType,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(secret))
}

// ParseToken validates a token string and returns its claims.
func ParseToken(tokenStr, secret string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
