package service

import (
	"context"
	"errors"
	"strings"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
)

var (
	// ErrEmployeeNoTaken is returned when employee_no is already registered.
	ErrEmployeeNoTaken = errors.New("service: employee_no already registered")
	// ErrEmployeeInactive is returned when the employee is deactivated.
	ErrEmployeeInactive = errors.New("service: employee is not active")
)

// EmployeeStore is the persistence contract EmployeeService needs.
type EmployeeStore interface {
	Create(ctx context.Context, tx repository.DBTX, e *model.Employee) error
	FindByID(ctx context.Context, q repository.DBTX, id int64) (*model.Employee, error)
	FindByEmployeeNo(ctx context.Context, q repository.DBTX, employeeNo string) (*model.Employee, error)
	List(ctx context.Context, q repository.DBTX, activeOnly bool, limit, offset int) ([]model.Employee, error)
	Update(ctx context.Context, tx repository.DBTX, e *model.Employee) error
	SoftDelete(ctx context.Context, tx repository.DBTX, id int64) error
}

// EmployeeService manages employee records.
type EmployeeService struct {
	db        repository.DBTX
	employees EmployeeStore
	transact  Transactor
}

// NewEmployeeService creates an EmployeeService.
func NewEmployeeService(db repository.DBTX, employees EmployeeStore, transact Transactor) *EmployeeService {
	return &EmployeeService{db: db, employees: employees, transact: transact}
}

// CreateEmployee registers a new employee. The duplicate check and the
// insert run in one transaction; the UNIQUE constraint on employee_no is
// the final guard against races.
func (s *EmployeeService) CreateEmployee(ctx context.Context, employeeNo, name, department string) (*model.Employee, error) {
	employeeNo = strings.TrimSpace(employeeNo)
	name = strings.TrimSpace(name)
	if employeeNo == "" {
		return nil, ValidationError{Field: "employee_no", Message: "employee_no is required"}
	}
	if name == "" {
		return nil, ValidationError{Field: "name", Message: "name is required"}
	}
	var created *model.Employee
	err := s.transact(ctx, func(tx repository.DBTX) error {
		if _, err := s.employees.FindByEmployeeNo(ctx, tx, employeeNo); err == nil {
			return ErrEmployeeNoTaken
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		e := &model.Employee{
			EmployeeNo: employeeNo,
			Name:       name,
			Department: strings.TrimSpace(department),
			IsActive:   true,
		}
		if err := s.employees.Create(ctx, tx, e); err != nil {
			return err
		}
		created = e
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ListEmployees returns a page of employees.
func (s *EmployeeService) ListEmployees(ctx context.Context, activeOnly bool, limit, offset int) ([]model.Employee, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.employees.List(ctx, s.db, activeOnly, limit, offset)
}
