package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

var (
	// ErrAlreadyCheckedIn is returned when checking in twice without a check-out.
	ErrAlreadyCheckedIn = errors.New("service: employee already checked in")
	// ErrNoOpenCheckIn is returned when checking out without an open check-in.
	ErrNoOpenCheckIn = errors.New("service: no open check-in found")
	// ErrEmployeeNotFound is returned when employee_no does not exist.
	ErrEmployeeNotFound = errors.New("service: employee not found")
	// ErrInvalidMethod is returned for an unknown attendance method.
	ErrInvalidMethod = errors.New("service: method must be 'face' or 'manual'")
)

// AttendanceStore is the persistence contract AttendanceService needs.
type AttendanceStore interface {
	CreateLog(ctx context.Context, tx repository.DBTX, l *model.AttendanceLog) error
	FindOpenCheckIn(ctx context.Context, q repository.DBTX, employeeID int64) (*model.AttendanceLog, error)
	ListByEmployee(ctx context.Context, q repository.DBTX, employeeID int64, from, to time.Time, limit, offset int) ([]model.AttendanceLog, error)
	GetEmployeeByNo(ctx context.Context, q repository.DBTX, employeeNo string) (*model.Employee, error)
	UpsertDevice(ctx context.Context, tx repository.DBTX, d *model.Device) error
}

// AttendanceService handles check-in, check-out and history.
type AttendanceService struct {
	db       repository.DBTX
	store    AttendanceStore
	transact Transactor
}

// NewAttendanceService creates an AttendanceService.
func NewAttendanceService(db repository.DBTX, store AttendanceStore, transact Transactor) *AttendanceService {
	return &AttendanceService{db: db, store: store, transact: transact}
}

// CheckIn records a check-in for the employee identified by employeeNo.
// Milestone 1 note: face verification arrives in Milestone 2; manual
// check-ins are treated as operator-verified.
func (s *AttendanceService) CheckIn(ctx context.Context, employeeNo, method, deviceID, platform string) (*model.AttendanceLog, error) {
	employeeNo = strings.TrimSpace(employeeNo)
	if employeeNo == "" {
		return nil, ValidationError{Field: "employee_no", Message: "employee_no is required"}
	}
	if method == "" {
		method = model.MethodManual
	}
	if method != model.MethodManual && method != model.MethodFace {
		return nil, ErrInvalidMethod
	}

	var created *model.AttendanceLog
	err := s.transact(ctx, func(tx repository.DBTX) error {
		emp, err := s.store.GetEmployeeByNo(ctx, tx, employeeNo)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrEmployeeNotFound
		}
		if err != nil {
			return err
		}
		if !emp.IsActive {
			return ErrEmployeeInactive
		}
		if open, err := s.store.FindOpenCheckIn(ctx, tx, emp.ID); err == nil && open != nil {
			return ErrAlreadyCheckedIn
		} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		l := &model.AttendanceLog{
			EmployeeID: emp.ID,
			Type:       model.AttendanceCheckIn,
			Method:     method,
			Verified:   method == model.MethodManual,
			RecordedAt: time.Now(),
		}
		if err := s.store.CreateLog(ctx, tx, l); err != nil {
			return err
		}
		created = l
		if strings.TrimSpace(deviceID) != "" {
			empID := emp.ID
			if err := s.store.UpsertDevice(ctx, tx, &model.Device{
				DeviceID:   strings.TrimSpace(deviceID),
				EmployeeID: &empID,
				Platform:   strings.TrimSpace(platform),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// CheckOut records a check-out against the employee's open check-in.
func (s *AttendanceService) CheckOut(ctx context.Context, employeeNo, deviceID, platform string) (*model.AttendanceLog, error) {
	employeeNo = strings.TrimSpace(employeeNo)
	if employeeNo == "" {
		return nil, ValidationError{Field: "employee_no", Message: "employee_no is required"}
	}

	var created *model.AttendanceLog
	err := s.transact(ctx, func(tx repository.DBTX) error {
		emp, err := s.store.GetEmployeeByNo(ctx, tx, employeeNo)
		if errors.Is(err, repository.ErrNotFound) {
			return ErrEmployeeNotFound
		}
		if err != nil {
			return err
		}
		open, err := s.store.FindOpenCheckIn(ctx, tx, emp.ID)
		if errors.Is(err, repository.ErrNotFound) || open == nil {
			return ErrNoOpenCheckIn
		}
		if err != nil {
			return err
		}
		l := &model.AttendanceLog{
			EmployeeID: emp.ID,
			Type:       model.AttendanceCheckOut,
			Method:     open.Method,
			Verified:   true,
			RecordedAt: time.Now(),
		}
		if err := s.store.CreateLog(ctx, tx, l); err != nil {
			return err
		}
		created = l
		if strings.TrimSpace(deviceID) != "" {
			empID := emp.ID
			if err := s.store.UpsertDevice(ctx, tx, &model.Device{
				DeviceID:   strings.TrimSpace(deviceID),
				EmployeeID: &empID,
				Platform:   strings.TrimSpace(platform),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// History returns attendance logs for an employee in a time range.
func (s *AttendanceService) History(ctx context.Context, employeeID int64, from, to time.Time, limit, offset int) ([]model.AttendanceLog, error) {
	if employeeID <= 0 {
		return nil, ValidationError{Field: "employee_id", Message: "employee_id is required"}
	}
	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() {
		from = to.AddDate(0, 0, -30)
	}
	if !from.Before(to) {
		return nil, ValidationError{Field: "from", Message: "from must be before to"}
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.ListByEmployee(ctx, s.db, employeeID, from, to, limit, offset)
}
