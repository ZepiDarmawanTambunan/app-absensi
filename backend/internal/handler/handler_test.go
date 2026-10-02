package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/service"
)

// fakeEmployeeService implements handler.EmployeeService for tests.
type fakeEmployeeService struct {
	employee  *model.Employee
	employees []model.Employee
	err       error
}

func (f *fakeEmployeeService) CreateEmployee(_ context.Context, no, name, dept string) (*model.Employee, error) {
	return f.employee, f.err
}

func (f *fakeEmployeeService) ListEmployees(_ context.Context, _ bool, _, _ int) ([]model.Employee, error) {
	return f.employees, f.err
}

func TestEmployeeHandler_Create(t *testing.T) {
	e := &model.Employee{ID: 1, EmployeeNo: "EMP001", Name: "Budi", Department: "IT", IsActive: true}
	h := NewEmployeeHandler(&fakeEmployeeService{employee: e})

	t.Run("success returns 201", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/employees",
			bytes.NewBufferString(`{"employee_no":"EMP001","name":"Budi","department":"IT"}`))
		rec := httptest.NewRecorder()
		h.Create(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}
		data, errObj := decodeEnvelope(t, rec)
		if errObj != nil {
			t.Fatalf("expected no error, got %v", errObj)
		}
		if data["employee_no"] != "EMP001" {
			t.Fatalf("unexpected data: %v", data)
		}
	})

	t.Run("duplicate returns 409", func(t *testing.T) {
		dup := NewEmployeeHandler(&fakeEmployeeService{err: service.ErrEmployeeNoTaken})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/employees",
			bytes.NewBufferString(`{"employee_no":"EMP001","name":"Budi"}`))
		rec := httptest.NewRecorder()
		dup.Create(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", rec.Code)
		}
	})
}

func TestEmployeeHandler_List(t *testing.T) {
	h := NewEmployeeHandler(&fakeEmployeeService{employees: []model.Employee{
		{ID: 1, EmployeeNo: "EMP001", Name: "Budi"},
	}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/employees?active_only=true&limit=10", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("expected 1 employee in data, got %v", body["data"])
	}
}

// fakeAttendanceService implements handler.AttendanceService for tests.
type fakeAttendanceService struct {
	log  *model.AttendanceLog
	logs []model.AttendanceLog
	err  error
}

func (f *fakeAttendanceService) CheckIn(_ context.Context, _, _, _, _ string) (*model.AttendanceLog, error) {
	return f.log, f.err
}

func (f *fakeAttendanceService) CheckOut(_ context.Context, _, _, _ string) (*model.AttendanceLog, error) {
	return f.log, f.err
}

func (f *fakeAttendanceService) History(_ context.Context, _ int64, _, _ time.Time, _, _ int) ([]model.AttendanceLog, error) {
	return f.logs, f.err
}

func TestAttendanceHandler_CheckIn(t *testing.T) {
	l := &model.AttendanceLog{ID: 1, EmployeeID: 1, Type: model.AttendanceCheckIn, Method: model.MethodManual, Verified: true}
	h := NewAttendanceHandler(&fakeAttendanceService{log: l})

	t.Run("success returns 201", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/attendance/check-in",
			bytes.NewBufferString(`{"employee_no":"EMP001","method":"manual","device_id":"dev-1"}`))
		rec := httptest.NewRecorder()
		h.CheckIn(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}
		data, _ := decodeEnvelope(t, rec)
		if data["type"] != "check_in" {
			t.Fatalf("unexpected data: %v", data)
		}
	})

	t.Run("double check-in returns 409", func(t *testing.T) {
		dup := NewAttendanceHandler(&fakeAttendanceService{err: service.ErrAlreadyCheckedIn})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/attendance/check-in",
			bytes.NewBufferString(`{"employee_no":"EMP001"}`))
		rec := httptest.NewRecorder()
		dup.CheckIn(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", rec.Code)
		}
	})

	t.Run("unknown employee returns 404", func(t *testing.T) {
		nf := NewAttendanceHandler(&fakeAttendanceService{err: service.ErrEmployeeNotFound})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/attendance/check-in",
			bytes.NewBufferString(`{"employee_no":"NOPE"}`))
		rec := httptest.NewRecorder()
		nf.CheckIn(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestAttendanceHandler_CheckOut(t *testing.T) {
	t.Run("check-out without open check-in returns 404", func(t *testing.T) {
		h := NewAttendanceHandler(&fakeAttendanceService{err: service.ErrNoOpenCheckIn})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/attendance/check-out",
			bytes.NewBufferString(`{"employee_no":"EMP001"}`))
		rec := httptest.NewRecorder()
		h.CheckOut(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestAttendanceHandler_History(t *testing.T) {
	h := NewAttendanceHandler(&fakeAttendanceService{logs: []model.AttendanceLog{
		{ID: 1, EmployeeID: 1, Type: model.AttendanceCheckIn},
	}})

	t.Run("success returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/attendance?employee_id=1", nil)
		rec := httptest.NewRecorder()
		h.History(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("missing employee_id returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/attendance", nil)
		rec := httptest.NewRecorder()
		h.History(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("bad from date returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/attendance?employee_id=1&from=nope", nil)
		rec := httptest.NewRecorder()
		h.History(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}
