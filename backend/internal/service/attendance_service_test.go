package service

import (
	"context"
	"testing"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

// fakeAttendanceStore is an in-memory AttendanceStore for tests.
type fakeAttendanceStore struct {
	employees map[string]*model.Employee
	logs      []model.AttendanceLog
	nextID    int64
	devices   map[string]*model.Device
}

func newFakeAttendanceStore() *fakeAttendanceStore {
	return &fakeAttendanceStore{
		employees: map[string]*model.Employee{},
		devices:   map[string]*model.Device{},
	}
}

func (f *fakeAttendanceStore) seedEmployee(no string, active bool) {
	f.nextID++
	e := &model.Employee{ID: f.nextID, EmployeeNo: no, Name: "Test " + no, IsActive: active}
	f.employees[no] = e
}

func (f *fakeAttendanceStore) CreateLog(_ context.Context, _ repository.DBTX, l *model.AttendanceLog) error {
	f.nextID++
	l.ID = f.nextID
	f.logs = append(f.logs, *l)
	return nil
}

func (f *fakeAttendanceStore) FindOpenCheckIn(_ context.Context, _ repository.DBTX, employeeID int64) (*model.AttendanceLog, error) {
	var open *model.AttendanceLog
	for i := range f.logs {
		l := &f.logs[i]
		if l.EmployeeID != employeeID || l.Type != model.AttendanceCheckIn {
			continue
		}
		closed := false
		for j := range f.logs {
			o := &f.logs[j]
			if o.EmployeeID == employeeID && o.Type == model.AttendanceCheckOut &&
				!o.RecordedAt.Before(l.RecordedAt) {
				closed = true
				break
			}
		}
		if !closed && (open == nil || l.RecordedAt.After(open.RecordedAt)) {
			c := *l
			open = &c
		}
	}
	if open == nil {
		return nil, repository.ErrNotFound
	}
	return open, nil
}

func (f *fakeAttendanceStore) ListByEmployee(_ context.Context, _ repository.DBTX, employeeID int64, from, to time.Time, limit, offset int) ([]model.AttendanceLog, error) {
	out := []model.AttendanceLog{}
	for _, l := range f.logs {
		if l.EmployeeID == employeeID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeAttendanceStore) GetEmployeeByNo(_ context.Context, _ repository.DBTX, no string) (*model.Employee, error) {
	e, ok := f.employees[no]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return e, nil
}

func (f *fakeAttendanceStore) UpsertDevice(_ context.Context, _ repository.DBTX, d *model.Device) error {
	f.devices[d.DeviceID] = d
	return nil
}

func newTestAttendanceService(store *fakeAttendanceStore) *AttendanceService {
	return NewAttendanceService(nil, store, fakeTransact)
}

func TestAttendanceService_CheckIn(t *testing.T) {
	ctx := context.Background()

	t.Run("valid check-in defaults to manual and verified", func(t *testing.T) {
		store := newFakeAttendanceStore()
		store.seedEmployee("EMP001", true)
		svc := newTestAttendanceService(store)

		l, err := svc.CheckIn(ctx, "EMP001", "", "dev-1", "android")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.Type != model.AttendanceCheckIn || l.Method != model.MethodManual {
			t.Fatalf("unexpected log: %+v", l)
		}
		if !l.Verified {
			t.Fatal("manual check-in should be verified in Milestone 1")
		}
		if _, ok := store.devices["dev-1"]; !ok {
			t.Fatal("device was not recorded")
		}
	})

	t.Run("double check-in rejected", func(t *testing.T) {
		store := newFakeAttendanceStore()
		store.seedEmployee("EMP001", true)
		svc := newTestAttendanceService(store)

		if _, err := svc.CheckIn(ctx, "EMP001", "manual", "", ""); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CheckIn(ctx, "EMP001", "manual", "", ""); err != ErrAlreadyCheckedIn {
			t.Fatalf("expected ErrAlreadyCheckedIn, got %v", err)
		}
	})

	t.Run("unknown employee", func(t *testing.T) {
		store := newFakeAttendanceStore()
		svc := newTestAttendanceService(store)
		if _, err := svc.CheckIn(ctx, "NOPE", "manual", "", ""); err != ErrEmployeeNotFound {
			t.Fatalf("expected ErrEmployeeNotFound, got %v", err)
		}
	})

	t.Run("inactive employee", func(t *testing.T) {
		store := newFakeAttendanceStore()
		store.seedEmployee("EMP002", false)
		svc := newTestAttendanceService(store)
		if _, err := svc.CheckIn(ctx, "EMP002", "manual", "", ""); err != ErrEmployeeInactive {
			t.Fatalf("expected ErrEmployeeInactive, got %v", err)
		}
	})

	t.Run("invalid method", func(t *testing.T) {
		store := newFakeAttendanceStore()
		store.seedEmployee("EMP001", true)
		svc := newTestAttendanceService(store)
		if _, err := svc.CheckIn(ctx, "EMP001", "qr", "", ""); err != ErrInvalidMethod {
			t.Fatalf("expected ErrInvalidMethod, got %v", err)
		}
	})

	t.Run("empty employee_no", func(t *testing.T) {
		store := newFakeAttendanceStore()
		svc := newTestAttendanceService(store)
		if _, err := svc.CheckIn(ctx, "", "manual", "", ""); err == nil {
			t.Fatal("expected validation error")
		}
	})
}

func TestAttendanceService_CheckOut(t *testing.T) {
	ctx := context.Background()

	t.Run("valid check-out inherits method", func(t *testing.T) {
		store := newFakeAttendanceStore()
		store.seedEmployee("EMP001", true)
		svc := newTestAttendanceService(store)

		in, err := svc.CheckIn(ctx, "EMP001", "manual", "", "")
		if err != nil {
			t.Fatal(err)
		}
		// Ensure the check-out sorts after the check-in.
		in.RecordedAt = time.Now().Add(-time.Minute)
		store.logs[0] = *in

		out, err := svc.CheckOut(ctx, "EMP001", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Type != model.AttendanceCheckOut || out.Method != model.MethodManual {
			t.Fatalf("unexpected log: %+v", out)
		}
	})

	t.Run("check-out without check-in", func(t *testing.T) {
		store := newFakeAttendanceStore()
		store.seedEmployee("EMP001", true)
		svc := newTestAttendanceService(store)
		if _, err := svc.CheckOut(ctx, "EMP001", "", ""); err != ErrNoOpenCheckIn {
			t.Fatalf("expected ErrNoOpenCheckIn, got %v", err)
		}
	})

	t.Run("check-in again after check-out", func(t *testing.T) {
		store := newFakeAttendanceStore()
		store.seedEmployee("EMP001", true)
		svc := newTestAttendanceService(store)

		in, err := svc.CheckIn(ctx, "EMP001", "manual", "", "")
		if err != nil {
			t.Fatal(err)
		}
		in.RecordedAt = time.Now().Add(-time.Minute)
		store.logs[0] = *in
		if _, err := svc.CheckOut(ctx, "EMP001", "", ""); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CheckIn(ctx, "EMP001", "manual", "", ""); err != nil {
			t.Fatalf("expected check-in to succeed after check-out, got %v", err)
		}
	})
}

func TestAttendanceService_History(t *testing.T) {
	ctx := context.Background()
	store := newFakeAttendanceStore()
	store.seedEmployee("EMP001", true)
	svc := newTestAttendanceService(store)

	if _, err := svc.CheckIn(ctx, "EMP001", "manual", "", ""); err != nil {
		t.Fatal(err)
	}
	logs, err := svc.History(ctx, 1, time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if _, err := svc.History(ctx, 0, time.Time{}, time.Time{}, 0, 0); err == nil {
		t.Fatal("expected validation error for employee_id 0")
	}
}
