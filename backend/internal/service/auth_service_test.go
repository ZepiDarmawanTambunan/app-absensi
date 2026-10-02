package service

import (
	"context"
	"testing"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

// fakeUserStore is an in-memory UserStore for tests.
type fakeUserStore struct {
	byEmail map[string]*model.User
	byID    map[int64]*model.User
	nextID  int64
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{byEmail: map[string]*model.User{}, byID: map[int64]*model.User{}}
}

func (f *fakeUserStore) Create(_ context.Context, _ repository.DBTX, u *model.User) error {
	f.nextID++
	u.ID = f.nextID
	u.CreatedAt = time.Now()
	f.byEmail[u.Email] = u
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUserStore) FindByEmail(_ context.Context, _ repository.DBTX, email string) (*model.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserStore) FindByID(_ context.Context, _ repository.DBTX, id int64) (*model.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

// fakeTransact runs fn without a real database.
func fakeTransact(ctx context.Context, fn func(tx repository.DBTX) error) error {
	return fn(nil)
}

func newTestAuthService(store *fakeUserStore) *AuthService {
	return NewAuthService(nil, store, fakeTransact, "test-secret-key-0123456789", time.Minute, time.Hour)
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name     string
		seed     func(*fakeUserStore)
		inName   string
		inEmail  string
		inPass   string
		wantErr  error
		wantCode string // validation field when wantErr is ValidationError
	}{
		{
			name:    "valid registration",
			inName:  "Admin",
			inEmail: "admin@example.com",
			inPass:  "secret123",
		},
		{
			name: "duplicate email",
			seed: func(s *fakeUserStore) {
				s.byEmail["admin@example.com"] = &model.User{ID: 1, Email: "admin@example.com"}
			},
			inName:  "Admin 2",
			inEmail: "admin@example.com",
			inPass:  "secret123",
			wantErr: ErrEmailTaken,
		},
		{
			name:     "empty name",
			inName:   "",
			inEmail:  "a@example.com",
			inPass:   "secret123",
			wantCode: "name",
		},
		{
			name:     "invalid email",
			inName:   "Admin",
			inEmail:  "not-an-email",
			inPass:   "secret123",
			wantCode: "email",
		},
		{
			name:     "short password",
			inName:   "Admin",
			inEmail:  "a@example.com",
			inPass:   "short",
			wantCode: "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeUserStore()
			if tt.seed != nil {
				tt.seed(store)
			}
			svc := newTestAuthService(store)
			tokens, err := svc.Register(context.Background(), tt.inName, tt.inEmail, tt.inPass)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if tt.wantCode != "" {
				ve, ok := err.(ValidationError)
				if !ok {
					t.Fatalf("expected ValidationError, got %T (%v)", err, err)
				}
				if ve.Field != tt.wantCode {
					t.Fatalf("expected field %q, got %q", tt.wantCode, ve.Field)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens.AccessToken == "" || tokens.RefreshToken == "" {
				t.Fatal("expected non-empty token pair")
			}
			if tokens.ExpiresIn != 60 {
				t.Fatalf("expected expires_in 60, got %d", tokens.ExpiresIn)
			}
			// Password must be stored hashed, never plain.
			stored := store.byEmail[tt.inEmail]
			if stored == nil {
				t.Fatal("user was not stored")
			}
			if stored.PasswordHash == tt.inPass {
				t.Fatal("password stored in plain text")
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	store := newFakeUserStore()
	svc := newTestAuthService(store)
	if _, err := svc.Register(context.Background(), "Admin", "admin@example.com", "secret123"); err != nil {
		t.Fatalf("setup register failed: %v", err)
	}

	t.Run("valid login", func(t *testing.T) {
		tokens, err := svc.Login(context.Background(), "admin@example.com", "secret123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokens.AccessToken == "" {
			t.Fatal("expected access token")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		if _, err := svc.Login(context.Background(), "admin@example.com", "wrongpass"); err != ErrInvalidCredentials {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("unknown email", func(t *testing.T) {
		if _, err := svc.Login(context.Background(), "nobody@example.com", "secret123"); err != ErrInvalidCredentials {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})
}

func TestAuthService_Refresh(t *testing.T) {
	store := newFakeUserStore()
	svc := newTestAuthService(store)
	tokens, err := svc.Register(context.Background(), "Admin", "admin@example.com", "secret123")
	if err != nil {
		t.Fatalf("setup register failed: %v", err)
	}

	t.Run("valid refresh", func(t *testing.T) {
		pair, err := svc.Refresh(context.Background(), tokens.RefreshToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pair.AccessToken == "" || pair.RefreshToken == "" {
			t.Fatal("expected new token pair")
		}
	})

	t.Run("access token rejected as refresh", func(t *testing.T) {
		if _, err := svc.Refresh(context.Background(), tokens.AccessToken); err != ErrInvalidToken {
			t.Fatalf("expected ErrInvalidToken, got %v", err)
		}
	})

	t.Run("garbage token", func(t *testing.T) {
		if _, err := svc.Refresh(context.Background(), "not-a-token"); err != ErrInvalidToken {
			t.Fatalf("expected ErrInvalidToken, got %v", err)
		}
	})
}
