package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

// Anti-spoofing defaults (Milestone 3). All are tunable via environment;
// see config.Load. Calibrate the liveness threshold against the actual
// on-device liveness model before tightening it.
const (
	// DefaultLivenessThreshold is the minimum on-device liveness score
	// accepted for a verification. 0.70 is a common operating point for
	// passive on-device liveness models: strict enough to reject most
	// photo/video replay attacks, lenient enough for low-end cameras.
	DefaultLivenessThreshold = 0.70
	// DefaultMaxLivenessFailures is how many rejected liveness checks
	// within LivenessFailureWindow trigger a temporary lockout.
	// 5 tolerates a genuine user struggling with lighting while
	// stopping automated replay attempts.
	DefaultMaxLivenessFailures = 5
	// DefaultLivenessLockoutMinutes is how long face verification stays
	// locked after the failure budget is exhausted. 15 minutes is short
	// enough not to strand an employee for a whole shift.
	DefaultLivenessLockoutMinutes = 15
	// LivenessFailureWindow is the trailing window in which rejected
	// liveness checks are counted toward a lockout.
	LivenessFailureWindow = time.Hour
	// LivenessRejectReason is stored on every rejected liveness check.
	LivenessRejectReason = "liveness_too_low"
	// ChallengeTTL is how long an issued liveness challenge stays valid.
	ChallengeTTL = 2 * time.Minute
)

var (
	// ErrChallengeNotFound is returned for an unknown challenge id.
	ErrChallengeNotFound = errors.New("service: challenge not found")
	// ErrChallengeExpired is returned for a challenge past its TTL.
	ErrChallengeExpired = errors.New("service: challenge expired")
	// ErrChallengeUsed is returned when a challenge id is reused.
	ErrChallengeUsed = errors.New("service: challenge already used")
)

// FaceLockedError is returned when face verification is temporarily
// locked after repeated failed liveness checks. The handler maps it
// to HTTP 423 with the remaining wait time.
type FaceLockedError struct {
	RetryAfter time.Duration
}

// Error implements error.
func (e *FaceLockedError) Error() string {
	secs := int64(e.RetryAfter.Round(time.Second).Seconds())
	if secs < 1 {
		secs = 1
	}
	return fmt.Sprintf("face verification locked after repeated failed liveness checks; retry in %d seconds", secs)
}

// SpoofStore is the persistence contract the liveness pipeline needs.
type SpoofStore interface {
	Record(ctx context.Context, tx repository.DBTX, a *model.SpoofAttempt) error
	CountRecent(ctx context.Context, q repository.DBTX, employeeID int64, since time.Time) (int, *time.Time, error)
}

// LivenessOptions tunes the anti-spoofing enforcement of FaceService.
type LivenessOptions struct {
	Threshold      float64
	Required       bool
	MaxFailures    int
	LockoutMinutes int
	Challenges     *ChallengeStore
}

// DefaultLivenessOptions returns the recommended starting configuration.
func DefaultLivenessOptions() LivenessOptions {
	return LivenessOptions{
		Threshold:      DefaultLivenessThreshold,
		Required:       true,
		MaxFailures:    DefaultMaxLivenessFailures,
		LockoutMinutes: DefaultLivenessLockoutMinutes,
		Challenges:     NewChallengeStore(ChallengeTTL),
	}
}

// Challenge types the mobile app can be asked to perform.
const (
	ChallengeBlink    = "blink"
	ChallengeTurnHead = "turn_head"
)

// Challenge is a single-use liveness challenge issued to the mobile app.
// The app performs it on-device; the resulting liveness score is sent
// back with the verify call together with the challenge id.
type Challenge struct {
	ID        string `json:"challenge_id"`
	Type      string `json:"type"`
	ExpiresIn int64  `json:"expires_in"` // seconds until expiry
}

type challengeEntry struct {
	typ       string
	expiresAt time.Time
	used      bool
}

// ChallengeStore issues and validates single-use liveness challenges.
// It is in-memory only: sufficient for a single instance, but a
// multi-instance production deployment should use Redis instead.
type ChallengeStore struct {
	mu    sync.Mutex
	items map[string]*challengeEntry
	ttl   time.Duration
	now   func() time.Time
}

// NewChallengeStore creates a ChallengeStore with the given TTL.
func NewChallengeStore(ttl time.Duration) *ChallengeStore {
	return &ChallengeStore{items: make(map[string]*challengeEntry), ttl: ttl, now: time.Now}
}

// Issue creates a new challenge with a random type and id.
func (s *ChallengeStore) Issue() *Challenge {
	var b [16]byte
	_, _ = rand.Read(b[:])
	id := hex.EncodeToString(b[:])

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked()

	typ := ChallengeBlink
	if b[0]%2 == 1 {
		typ = ChallengeTurnHead
	}
	s.items[id] = &challengeEntry{typ: typ, expiresAt: s.now().Add(s.ttl)}
	return &Challenge{ID: id, Type: typ, ExpiresIn: int64(s.ttl.Seconds())}
}

// Consume validates a challenge id and marks it used. It returns
// ErrChallengeNotFound, ErrChallengeExpired, or ErrChallengeUsed.
func (s *ChallengeStore) Consume(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.items[id]
	if !ok {
		return ErrChallengeNotFound
	}
	now := s.now()
	if now.After(e.expiresAt) {
		delete(s.items, id)
		return ErrChallengeExpired
	}
	if e.used {
		return ErrChallengeUsed
	}
	e.used = true
	return nil
}

// sweepLocked drops expired entries. Callers must hold s.mu.
func (s *ChallengeStore) sweepLocked() {
	now := s.now()
	for id, e := range s.items {
		if now.After(e.expiresAt) {
			delete(s.items, id)
		}
	}
}
