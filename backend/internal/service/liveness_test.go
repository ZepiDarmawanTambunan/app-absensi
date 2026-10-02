package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

// fakeSpoofStore is an in-memory SpoofStore for liveness tests.
type fakeSpoofStore struct {
	attempts []model.SpoofAttempt
	now      func() time.Time
}

func newFakeSpoofStore() *fakeSpoofStore {
	return &fakeSpoofStore{now: time.Now}
}

func (f *fakeSpoofStore) Record(_ context.Context, _ repository.DBTX, a *model.SpoofAttempt) error {
	a.ID = int64(len(f.attempts)) + 1
	a.CreatedAt = f.now()
	f.attempts = append(f.attempts, *a)
	return nil
}

func (f *fakeSpoofStore) CountRecent(_ context.Context, _ repository.DBTX, employeeID int64, since time.Time) (int, *time.Time, error) {
	n := 0
	var last *time.Time
	for i := range f.attempts {
		a := &f.attempts[i]
		if a.EmployeeID != employeeID || a.CreatedAt.Before(since) {
			continue
		}
		n++
		if last == nil || a.CreatedAt.After(*last) {
			t := a.CreatedAt
			last = &t
		}
	}
	return n, last, nil
}

// livenessFixture builds a FaceService with a controllable clock shared
// between the service and the fake spoof store. It returns the service,
// the spoof store, and an advance function that moves the clock forward.
func livenessFixture() (*FaceService, *fakeSpoofStore, func(time.Duration)) {
	emps := &fakeFaceEmployeeStore{employees: map[int64]*model.Employee{
		1: {ID: 1, EmployeeNo: "EMP001", Name: "Budi", IsActive: true},
	}}
	faces := newFakeFaceStore()
	spoofs := newFakeSpoofStore()
	clock := time.Now()
	nowFn := func() time.Time { return clock }
	spoofs.now = nowFn
	svc := NewFaceService(nil, emps, faces, spoofs, fakeFaceTransact, DefaultMatchThreshold, DefaultLivenessOptions())
	svc.now = nowFn
	advance := func(d time.Duration) { clock = clock.Add(d) }
	return svc, spoofs, advance
}

func enrollLiveness(t *testing.T, svc *FaceService) {
	t.Helper()
	if _, err := svc.Enroll(context.Background(), 1,
		[][]float32{vec128(3, 0), vec128(0, 4)}, []float64{0.9, 0.8}); err != nil {
		t.Fatal(err)
	}
}

func TestLiveness_BelowThresholdRejectedAndLogged(t *testing.T) {
	ctx := context.Background()
	svc, spoofs, _ := livenessFixture()
	enrollLiveness(t, svc)

	// Probe would match the template head (0.6, 0.8), but liveness fails.
	score := 0.30
	res, err := svc.Verify(ctx, VerifyInput{
		EmployeeID: 1, Embedding: vec128(0.61, 0.79),
		LivenessScore: &score, DeviceID: "dev-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Match {
		t.Fatal("expected match=false on failed liveness")
	}
	if res.RejectReason != LivenessRejectReason {
		t.Fatalf("expected reject_reason %q, got %q", LivenessRejectReason, res.RejectReason)
	}
	if res.Distance != 0 {
		t.Fatalf("embedding must not be compared on liveness failure, distance=%v", res.Distance)
	}
	if res.Liveness == nil || res.Liveness.Passed || res.Liveness.Score != 0.30 {
		t.Fatalf("unexpected liveness info: %+v", res.Liveness)
	}
	if res.Liveness.Threshold != DefaultLivenessThreshold {
		t.Fatalf("expected liveness threshold %v, got %v", DefaultLivenessThreshold, res.Liveness.Threshold)
	}

	if len(spoofs.attempts) != 1 {
		t.Fatalf("expected 1 spoof attempt logged, got %d", len(spoofs.attempts))
	}
	a := spoofs.attempts[0]
	if a.EmployeeID != 1 || a.LivenessScore != 0.30 || a.Reason != LivenessRejectReason {
		t.Fatalf("unexpected spoof attempt: %+v", a)
	}
	if a.DeviceID == nil || *a.DeviceID != "dev-1" {
		t.Fatalf("expected device_id dev-1 logged, got %+v", a.DeviceID)
	}
}

func TestLiveness_DisabledAllowsMissingScore(t *testing.T) {
	ctx := context.Background()
	emps := &fakeFaceEmployeeStore{employees: map[int64]*model.Employee{
		1: {ID: 1, EmployeeNo: "EMP001", Name: "Budi", IsActive: true},
	}}
	faces := newFakeFaceStore()
	spoofs := newFakeSpoofStore()
	opts := DefaultLivenessOptions()
	opts.Required = false
	svc := NewFaceService(nil, emps, faces, spoofs, fakeFaceTransact, DefaultMatchThreshold, opts)
	enrollLiveness(t, svc)

	// No score at all: allowed, no liveness info attached.
	res, err := svc.Verify(ctx, VerifyInput{EmployeeID: 1, Embedding: vec128(0.61, 0.79)})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Match {
		t.Fatal("expected match")
	}
	if res.Liveness != nil {
		t.Fatalf("expected no liveness info, got %+v", res.Liveness)
	}

	// A supplied score is still validated and reported.
	good := 0.95
	res, err = svc.Verify(ctx, VerifyInput{EmployeeID: 1, Embedding: vec128(0.61, 0.79), LivenessScore: &good})
	if err != nil {
		t.Fatal(err)
	}
	if res.Liveness == nil || !res.Liveness.Passed {
		t.Fatalf("expected passed liveness info, got %+v", res.Liveness)
	}
}

func TestLiveness_LockoutAfterRepeatedFailures(t *testing.T) {
	ctx := context.Background()
	svc, _, advance := livenessFixture()
	enrollLiveness(t, svc)

	bad := 0.10
	verifyBad := func() (*VerifyResult, error) {
		return svc.Verify(ctx, VerifyInput{EmployeeID: 1, Embedding: vec128(0.61, 0.79), LivenessScore: &bad})
	}

	// Exhaust the failure budget; each attempt is rejected, not locked.
	for i := 0; i < DefaultMaxLivenessFailures; i++ {
		res, err := verifyBad()
		if err != nil {
			t.Fatalf("attempt %d: unexpected error %v", i, err)
		}
		if res.Match {
			t.Fatalf("attempt %d: expected rejection", i)
		}
		advance(time.Minute)
	}

	// The next attempt is locked even with a perfect liveness score.
	good := 0.99
	_, err := svc.Verify(ctx, VerifyInput{EmployeeID: 1, Embedding: vec128(0.61, 0.79), LivenessScore: &good})
	var lockedErr *FaceLockedError
	if !errors.As(err, &lockedErr) {
		t.Fatalf("expected FaceLockedError, got %v", err)
	}
	if lockedErr.RetryAfter <= 0 || lockedErr.RetryAfter > time.Duration(DefaultLivenessLockoutMinutes)*time.Minute {
		t.Fatalf("unexpected retry_after %v", lockedErr.RetryAfter)
	}

	locked, retryAfter, err := svc.IsFaceLocked(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !locked || retryAfter <= 0 {
		t.Fatalf("expected locked=true with positive retry_after, got %v %v", locked, retryAfter)
	}

	// Other employees are not affected.
	locked, _, err = svc.IsFaceLocked(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Fatal("lockout must be per-employee")
	}

	// After the lockout period passes, verification works again automatically.
	advance(time.Duration(DefaultLivenessLockoutMinutes+1) * time.Minute)
	locked, _, err = svc.IsFaceLocked(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Fatal("lockout should lift automatically after the lockout period")
	}
	res, err := svc.Verify(ctx, VerifyInput{EmployeeID: 1, Embedding: vec128(0.61, 0.79), LivenessScore: &good})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Match {
		t.Fatal("expected match after lockout lifted")
	}
}

func TestLiveness_OldFailuresAgeOut(t *testing.T) {
	ctx := context.Background()
	svc, _, advance := livenessFixture()
	enrollLiveness(t, svc)

	// Failures older than the window do not count toward a lockout.
	advance(-(LivenessFailureWindow + time.Minute))
	bad := 0.10
	for i := 0; i < DefaultMaxLivenessFailures; i++ {
		if _, err := svc.Verify(ctx, VerifyInput{EmployeeID: 1, Embedding: vec128(0, 1), LivenessScore: &bad}); err != nil {
			t.Fatal(err)
		}
	}
	// Back to "now": the old failures are outside the window.
	advance(LivenessFailureWindow + 2*time.Minute)

	locked, _, err := svc.IsFaceLocked(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Fatal("failures outside the window must not lock the account")
	}
}

func TestChallenge_IssueAndConsume(t *testing.T) {
	store := NewChallengeStore(2 * time.Minute)

	c := store.Issue()
	if c.ID == "" {
		t.Fatal("expected non-empty challenge id")
	}
	if c.Type != ChallengeBlink && c.Type != ChallengeTurnHead {
		t.Fatalf("unexpected challenge type %q", c.Type)
	}
	if c.ExpiresIn != 120 {
		t.Fatalf("expected expires_in 120, got %d", c.ExpiresIn)
	}

	if err := store.Consume(c.ID); err != nil {
		t.Fatalf("expected consume to succeed, got %v", err)
	}
	if err := store.Consume(c.ID); !errors.Is(err, ErrChallengeUsed) {
		t.Fatalf("expected ErrChallengeUsed, got %v", err)
	}
	if err := store.Consume("does-not-exist"); !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestChallenge_Expiry(t *testing.T) {
	store := NewChallengeStore(2 * time.Minute)
	now := time.Now()
	store.now = func() time.Time { return now }

	c := store.Issue()
	now = now.Add(3 * time.Minute)
	if err := store.Consume(c.ID); !errors.Is(err, ErrChallengeExpired) {
		t.Fatalf("expected ErrChallengeExpired, got %v", err)
	}
}

func TestLiveness_ChallengeEnforcedOnVerify(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := livenessFixture()
	enrollLiveness(t, svc)
	good := 0.95

	ch := svc.IssueChallenge()
	res, err := svc.Verify(ctx, VerifyInput{
		EmployeeID: 1, Embedding: vec128(0.61, 0.79),
		LivenessScore: &good, ChallengeID: ch.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Match {
		t.Fatal("expected match with valid challenge")
	}

	// Reusing the same challenge is rejected (replay protection).
	_, err = svc.Verify(ctx, VerifyInput{
		EmployeeID: 1, Embedding: vec128(0.61, 0.79),
		LivenessScore: &good, ChallengeID: ch.ID,
	})
	if !errors.Is(err, ErrChallengeUsed) {
		t.Fatalf("expected ErrChallengeUsed, got %v", err)
	}

	// Unknown challenge id is rejected.
	_, err = svc.Verify(ctx, VerifyInput{
		EmployeeID: 1, Embedding: vec128(0.61, 0.79),
		LivenessScore: &good, ChallengeID: "bogus",
	})
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}
