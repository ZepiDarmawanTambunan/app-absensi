package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
)

// EmployeeService is the contract EmployeeHandler needs.
type EmployeeService interface {
	CreateEmployee(ctx context.Context, employeeNo, name, department string) (*model.Employee, error)
	ListEmployees(ctx context.Context, activeOnly bool, limit, offset int) ([]model.Employee, error)
}

// EmployeeHandler serves the /employees endpoints.
type EmployeeHandler struct {
	svc EmployeeService
}

// NewEmployeeHandler creates an EmployeeHandler.
func NewEmployeeHandler(svc EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

type createEmployeeRequest struct {
	EmployeeNo string `json:"employee_no"`
	Name       string `json:"name"`
	Department string `json:"department"`
}

// Create handles POST /api/v1/employees.
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEmployeeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	e, err := h.svc.CreateEmployee(r.Context(), req.EmployeeNo, req.Name, req.Department)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// List handles GET /api/v1/employees.
func (h *EmployeeHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	activeOnly := q.Get("active_only") == "true"
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	employees, err := h.svc.ListEmployees(r.Context(), activeOnly, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if employees == nil {
		employees = []model.Employee{}
	}
	writeJSON(w, http.StatusOK, employees)
}
