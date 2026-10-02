package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

// fakeFaceEmployeeStore is a minimal in-memory EmployeeStore for face tests.
type fakeFaceEmployeeStore struct {
	employees map[int64]*model.Employee
}

func (f *fakeFaceEmployeeStore) Create(_ context.Context, _ repository.DBTX, e *model.Employee) error {
	return nil
}
func (f *fakeFaceEmployeeStore) FindByID(_ context.Context, _ repository.DBTX, id int64) (*model.Employee, error) {
	e, ok := f.employees[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return e, nil
}
func (f *fakeFaceEmployeeStore) FindByEmployeeNo(_ context.Context, _ repository.DBTX, _ string) (*model.Employee, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeFaceEmployeeStore) List(_ context.Context, _ repository.DBTX, _ bool, _, _ int) ([]model.Employee, error) {
	return nil, nil
}
func (f *fakeFaceEmployeeStore) Update(_ context.Context, _ repository.DBTX, _ *model.Employee) error {
	return nil
}
func (f *fakeFaceEmployeeStore) SoftDelete(_ context.Context, _ repository.DBTX, _ int64) error {
	return nil
}

// fakeFaceStore is an in-memory FaceStore for tests.
type fakeFaceStore struct {
	templates    map[int64]*model.FaceEnrollment
	replaceCalls int
}

func newFakeFaceStore() *fakeFaceStore {
	return &fakeFaceStore{templates: map[int64]*model.FaceEnrollment{}}
}

func (f *fakeFaceStore) Replace(_ context.Context, _ repository.DBTX, e *model.FaceEnrollment) error {
	f.replaceCalls++
	e.ID = int64(f.replaceCalls)
	c := *e
	f.templates[e.EmployeeID] = &c
	return nil
}

func (f *fakeFaceStore) FindByEmployeeID(_ context.Context, _ repository.DBTX, employeeID int64) (*model.FaceEnrollment, error) {
	e, ok := f.templates[employeeID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return e, nil
}

func fakeFaceTransact(ctx context.Context, fn func(tx repository.DBTX) error) error {
	return fn(nil)
}

func newFaceServiceFixture() (*FaceService, *fakeFaceEmployeeStore, *fakeFaceStore) {
	emps := &fakeFaceEmployeeStore{employees: map[int64]*model.Employee{
		1: {ID: 1, EmployeeNo: "EMP001", Name: "Budi", IsActive: true},
		2: {ID: 2, EmployeeNo: "EMP002", Name: "Siti", IsActive: false},
	}}
	faces := newFakeFaceStore()
	svc := NewFaceService(nil, emps, faces, fakeFaceTransact, DefaultMatchThreshold)
	return svc, emps, faces
}

// vec128 builds a 128-dim embedding with v at index 0 and w at index 1.
func vec128(v, w float32) []float32 {
	e := make([]float32, 128)
	e[0] = v
	e[1] = w
	return e
}

func isValidationErr(t *testing.T, err error) {
	t.Helper()
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestFaceService_Enroll(t *testing.T) {
	ctx := context.Background()

	t.Run("success averages quality-passing frames", func(t *testing.T) {
		svc, _, faces := newFaceServiceFixture()
		embs := [][]float32{vec128(3, 0), vec128(0, 0), vec128(0, 4)}
		quals := []float64{0.9, 0.4, 0.8} // middle frame discarded
		e, err := svc.Enroll(ctx, 1, embs, quals)
		if err != nil {
			t.Fatal(err)
		}
		if faces.replaceCalls != 1 {
			t.Fatalf("expected 1 Replace call, got %d", faces.replaceCalls)
		}
		if e.Dimension != 128 {
			t.Fatalf("expected dimension 128, got %d", e.Dimension)
		}
		// Average of (3,0) and (0,4) is (1.5,2), normalized -> (0.6,0.8).
		if len(e.Embedding) != 128 || e.Embedding[0] != 0.6 || e.Embedding[1] != 0.8 {
			t.Fatalf("unexpected template head: %v %v", e.Embedding[0], e.Embedding[1])
		}
		if e.QualityScore < 0.849 || e.QualityScore > 0.851 {
			t.Fatalf("expected quality ~0.85, got %v", e.QualityScore)
		}
	})

	t.Run("dimension mismatch returns 400-class error", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		_, err := svc.Enroll(ctx, 1, [][]float32{vec128(1, 0), make([]float32, 192)}, []float64{0.9, 0.9})
		isValidationErr(t, err)
	})

	t.Run("unsupported dimension rejected", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		_, err := svc.Enroll(ctx, 1, [][]float32{make([]float32, 64)}, []float64{0.9})
		isValidationErr(t, err)
	})

	t.Run("misaligned quality scores rejected", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		_, err := svc.Enroll(ctx, 1, [][]float32{vec128(1, 0)}, []float64{0.9, 0.8})
		isValidationErr(t, err)
	})

	t.Run("all frames below quality bar rejected", func(t *testing.T) {
		svc, _, faces := newFaceServiceFixture()
		_, err := svc.Enroll(ctx, 1, [][]float32{vec128(1, 0)}, []float64{0.1})
		isValidationErr(t, err)
		if faces.replaceCalls != 0 {
			t.Fatal("Replace must not be called")
		}
	})

	t.Run("unknown employee returns 404-class error", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		_, err := svc.Enroll(ctx, 99, [][]float32{vec128(1, 0)}, []float64{0.9})
		if !errors.Is(err, ErrEmployeeNotFound) {
			t.Fatalf("expected ErrEmployeeNotFound, got %v", err)
		}
	})

	t.Run("inactive employee rejected", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		_, err := svc.Enroll(ctx, 2, [][]float32{vec128(1, 0)}, []float64{0.9})
		if !errors.Is(err, ErrEmployeeInactive) {
			t.Fatalf("expected ErrEmployeeInactive, got %v", err)
		}
	})

	t.Run("re-enroll replaces previous template", func(t *testing.T) {
		svc, _, faces := newFaceServiceFixture()
		if _, err := svc.Enroll(ctx, 1, [][]float32{vec128(1, 0)}, []float64{0.9}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Enroll(ctx, 1, [][]float32{vec128(0, 1)}, []float64{0.9}); err != nil {
			t.Fatal(err)
		}
		if faces.replaceCalls != 2 {
			t.Fatalf("expected 2 Replace calls, got %d", faces.replaceCalls)
		}
		if len(faces.templates) != 1 {
			t.Fatalf("expected 1 stored template, got %d", len(faces.templates))
		}
	})
}

func TestFaceService_Verify(t *testing.T) {
	ctx := context.Background()
	enroll := func(t *testing.T, svc *FaceService) {
		t.Helper()
		if _, err := svc.Enroll(ctx, 1, [][]float32{vec128(3, 0), vec128(0, 4)}, []float64{0.9, 0.8}); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("matching probe returns match=true", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		enroll(t, svc)
		// Template head is (0.6, 0.8); probe close to it.
		res, err := svc.Verify(ctx, 1, vec128(0.61, 0.79), nil)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Match {
			t.Fatalf("expected match, distance=%v", res.Distance)
		}
		if res.Threshold != DefaultMatchThreshold {
			t.Fatalf("expected threshold %v, got %v", DefaultMatchThreshold, res.Threshold)
		}
	})

	t.Run("distant probe returns match=false", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		enroll(t, svc)
		// Orthogonal-ish probe: template head (0.6,0.8), probe head (-0.8,0.6).
		res, err := svc.Verify(ctx, 1, vec128(-0.8, 0.6), nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.Match {
			t.Fatalf("expected no match, distance=%v", res.Distance)
		}
		if res.Distance < res.Threshold {
			t.Fatalf("distance %v should exceed threshold %v", res.Distance, res.Threshold)
		}
	})

	t.Run("employee without enrollment returns 404-class error", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		_, err := svc.Verify(ctx, 1, vec128(1, 0), nil)
		if !errors.Is(err, ErrNoFaceEnrollment) {
			t.Fatalf("expected ErrNoFaceEnrollment, got %v", err)
		}
	})

	t.Run("unknown employee returns 404-class error", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		_, err := svc.Verify(ctx, 99, vec128(1, 0), nil)
		if !errors.Is(err, ErrEmployeeNotFound) {
			t.Fatalf("expected ErrEmployeeNotFound, got %v", err)
		}
	})

	t.Run("probe dimension mismatch rejected", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		enroll(t, svc)
		_, err := svc.Verify(ctx, 1, make([]float32, 512), nil)
		isValidationErr(t, err)
	})

	t.Run("out-of-range liveness score rejected", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		enroll(t, svc)
		bad := 1.5
		_, err := svc.Verify(ctx, 1, vec128(0.6, 0.8), &bad)
		isValidationErr(t, err)
	})

	t.Run("liveness score accepted and ignored for now", func(t *testing.T) {
		svc, _, _ := newFaceServiceFixture()
		enroll(t, svc)
		ok := 0.92
		res, err := svc.Verify(ctx, 1, vec128(0.61, 0.79), &ok)
		if err != nil {
			t.Fatal(err)
		}
		if !res.Match {
			t.Fatal("expected match")
		}
	})

	t.Run("invalid threshold falls back to default", func(t *testing.T) {
		_, emps, faces := newFaceServiceFixture()
		svc := NewFaceService(nil, emps, faces, fakeFaceTransact, -1)
		if svc.matchThreshold != DefaultMatchThreshold {
			t.Fatalf("expected default threshold, got %v", svc.matchThreshold)
		}
	})
}
