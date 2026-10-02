package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
)

// AttendanceService is the contract AttendanceHandler needs.
type AttendanceService interface {
	CheckIn(ctx context.Context, employeeNo, method, deviceID, platform string) (*model.AttendanceLog, error)
	CheckOut(ctx context.Context, employeeNo, deviceID, platform string) (*model.AttendanceLog, error)
	History(ctx context.Context, employeeID int64, from, to time.Time, limit, offset int) ([]model.AttendanceLog, error)
}

// AttendanceHandler serves the /attendance endpoints.
type AttendanceHandler struct {
	svc AttendanceService
}

// NewAttendanceHandler creates an AttendanceHandler.
func NewAttendanceHandler(svc AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{svc: svc}
}

type checkInRequest struct {
	EmployeeNo string `json:"employee_no"`
	Method     string `json:"method"`
	DeviceID   string `json:"device_id"`
	Platform   string `json:"platform"`
}

type checkOutRequest struct {
	EmployeeNo string `json:"employee_no"`
	DeviceID   string `json:"device_id"`
	Platform   string `json:"platform"`
}

// CheckIn handles POST /api/v1/attendance/check-in.
func (h *AttendanceHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	var req checkInRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	l, err := h.svc.CheckIn(r.Context(), req.EmployeeNo, req.Method, req.DeviceID, req.Platform)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

// CheckOut handles POST /api/v1/attendance/check-out.
func (h *AttendanceHandler) CheckOut(w http.ResponseWriter, r *http.Request) {
	var req checkOutRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	l, err := h.svc.CheckOut(r.Context(), req.EmployeeNo, req.DeviceID, req.Platform)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

// History handles GET /api/v1/attendance?employee_id=...
func (h *AttendanceHandler) History(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	employeeID, err := strconv.ParseInt(q.Get("employee_id"), 10, 64)
	if err != nil || employeeID <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "employee_id query parameter is required")
		return
	}
	var from, to time.Time
	if v := q.Get("from"); v != "" {
		from, err = time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "from must be RFC3339")
			return
		}
	}
	if v := q.Get("to"); v != "" {
		to, err = time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "to must be RFC3339")
			return
		}
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	logs, err := h.svc.History(r.Context(), employeeID, from, to, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if logs == nil {
		logs = []model.AttendanceLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}
