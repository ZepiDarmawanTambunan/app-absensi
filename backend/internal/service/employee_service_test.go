package service

import (
	"context"
	"testing"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

// fakeEmployeeStore is an in-memory EmployeeStore for tests.
type fakeEmployeeStore struct {
	byNo   map[string]*model.Employee
	byID   map[int64]*model.Employee
	nextID int64
}

func newFakeEmployeeStore() *fakeEmployeeStore {
	return &fakeEmployeeStore{byNo: map[string]*model.Employee{}, byID: map[int64]*model.Employee{}}
}

func (f *fakeEmployeeStore) Create(_ context.Context, _ repository.DBTX, e *model.Employee) error {
	f.nextID++
	e.ID = f.nextID
	f.byNo[e.EmployeeNo] = e
	f.byID[e.ID] = e
	return nil
}

func (f *fakeEmployeeStore) FindByID(_ context.Context, _ repository.DBTX, id int64) (*model.Employee, error) {
	e, ok := f.byID[id]
	if !ok || e.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	return e, nil
}

func (f *fakeEmployeeStore) FindByEmployeeNo(_ context.Context, _ repository.DBTX, no string) (*model.Employee, error) {
	e, ok := f.byNo[no]
	if !ok || e.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	return e, nil
}

func (f *fakeEmployeeStore) List(_ context.Context, _ repository.DBTX, activeOnly bool, limit, offset int) ([]model.Employee, error) {
	out := []model.Employee{}
	for _, e := range f.byID {
		if e.DeletedAt != nil {
			continue
		}
		if activeOnly && !e.IsActive {
			continue
		}
		out = append(out, *e)
	}
	return out, nil
}

func (f *fakeEmployeeStore) Update(_ context.Context, _ repository.DBTX, e *model.Employee) error {
	return nil
}

func (f *fakeEmployeeStore) SoftDelete(_ context.Context, _ repository.DBTX, id int64) error {
	return nil
}

func newTestEmployeeService(store *fakeEmployeeStore) *EmployeeService {
	return NewEmployeeService(nil, store, fakeTransact)
}

func TestEmployeeService_CreateEmployee(t *testing.T) {
	tests := []struct {
		name    string
		no      string
		empName string
		dept    string
		wantErr error
		wantFld string
	}{
		{name: "valid", no: "EMP001", empName: "Budi", dept: "IT"},
		{name: "duplicate employee_no", no: "EMP001", empName: "Budi 2", wantErr: ErrEmployeeNoTaken},
		{name: "empty employee_no", no: "", empName: "Budi", wantFld: "employee_no"},
		{name: "empty name", no: "EMP002", empName: "", wantFld: "name"},
	}

	store := newFakeEmployeeStore()
	svc := newTestEmployeeService(store)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := svc.CreateEmployee(context.Background(), tt.no, tt.empName, tt.dept)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if tt.wantFld != "" {
				ve, ok := err.(ValidationError)
				if !ok || ve.Field != tt.wantFld {
					t.Fatalf("expected ValidationError for %q, got %v", tt.wantFld, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if e.ID == 0 || !e.IsActive {
				t.Fatalf("expected persisted active employee, got %+v", e)
			}
		})
	}
}

func TestEmployeeService_ListEmployees(t *testing.T) {
	store := newFakeEmployeeStore()
	svc := newTestEmployeeService(store)
	ctx := context.Background()

	if _, err := svc.CreateEmployee(ctx, "EMP001", "Budi", "IT"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateEmployee(ctx, "EMP002", "Siti", "HR"); err != nil {
		t.Fatal(err)
	}

	all, err := svc.ListEmployees(ctx, false, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(all))
	}

	active, err := svc.ListEmployees(ctx, true, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active employees, got %d", len(active))
	}
}
