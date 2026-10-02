package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/auth"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

var (
	// ErrInvalidCredentials is returned when email/password do not match.
	ErrInvalidCredentials = errors.New("service: invalid email or password")
	// ErrEmailTaken is returned when registering an existing email.
	ErrEmailTaken = errors.New("service: email already registered")
	// ErrInvalidToken is returned when a refresh token is not valid.
	ErrInvalidToken = errors.New("service: invalid or expired token")
)

// UserStore is the persistence contract AuthService needs.
type UserStore interface {
	Create(ctx context.Context, tx repository.DBTX, u *model.User) error
	FindByEmail(ctx context.Context, q repository.DBTX, email string) (*model.User, error)
	FindByID(ctx context.Context, q repository.DBTX, id int64) (*model.User, error)
}

// AuthService handles registration, login and token refresh.
type AuthService struct {
	db         repository.DBTX
	users      UserStore
	transact   Transactor
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewAuthService creates an AuthService. In production pass a *sql.DB as db.
func NewAuthService(db repository.DBTX, users UserStore, transact Transactor, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		db:         db,
		users:      users,
		transact:   transact,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Register creates a new operator account and returns a token pair.
func (s *AuthService) Register(ctx context.Context, name, email, password string) (*model.TokenPair, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ValidationError{Field: "name", Message: "name is required"}
	}
	if !validEmail(email) {
		return nil, ValidationError{Field: "email", Message: "email is not valid"}
	}
	if len(password) < 8 {
		return nil, ValidationError{Field: "password", Message: "password must be at least 8 characters"}
	}
	email = normalizeEmail(email)

	if _, err := s.users.FindByEmail(ctx, s.db, email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &model.User{
		Name:         strings.TrimSpace(name),
		Email:        email,
		PasswordHash: string(hash),
		Role:         "admin",
	}
	if err := s.transact(ctx, func(tx repository.DBTX) error {
		return s.users.Create(ctx, tx, u)
	}); err != nil {
		return nil, err
	}
	return s.issueTokens(u)
}

// Login verifies credentials and returns a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (*model.TokenPair, error) {
	u, err := s.users.FindByEmail(ctx, s.db, normalizeEmail(email))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(u)
}

// Refresh exchanges a valid refresh token for a new token pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	claims, err := auth.ParseToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != auth.TokenTypeRefresh {
		return nil, ErrInvalidToken
	}
	u, err := s.users.FindByID(ctx, s.db, claims.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	return s.issueTokens(u)
}

func (s *AuthService) issueTokens(u *model.User) (*model.TokenPair, error) {
	access, err := auth.IssueToken(u, s.jwtSecret, auth.TokenTypeAccess, s.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := auth.IssueToken(u, s.jwtSecret, auth.TokenTypeRefresh, s.refreshTTL)
	if err != nil {
		return nil, err
	}
	return &model.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}
