// Package model defines the domain entities of the attendance system.
package model

import "time"

// Attendance types.
const (
	AttendanceCheckIn  = "check_in"
	AttendanceCheckOut = "check_out"
)

// Attendance methods. MethodFace is reserved for Milestone 2.
const (
	MethodFace   = "face"
	MethodManual = "manual"
)

// User is an operator account that can call the API.
type User struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Employee is a staff member whose attendance is tracked.
type Employee struct {
	ID         int64      `json:"id"`
	EmployeeNo string     `json:"employee_no"`
	Name       string     `json:"name"`
	Department string     `json:"department"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

// Device is a phone/tablet used for check-in.
type Device struct {
	ID         int64     `json:"id"`
	DeviceID   string    `json:"device_id"`
	EmployeeID *int64    `json:"employee_id,omitempty"`
	Platform   string    `json:"platform"`
	LastSeen   time.Time `json:"last_seen"`
}

// AttendanceLog is a single check-in or check-out event.
type AttendanceLog struct {
	ID            int64     `json:"id"`
	EmployeeID    int64     `json:"employee_id"`
	Type          string    `json:"type"`
	Method        string    `json:"method"`
	Verified      bool      `json:"verified"`
	LivenessScore *float64  `json:"liveness_score,omitempty"`
	RecordedAt    time.Time `json:"recorded_at"`
}

// TokenPair is issued on register/login/refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds until the access token expires
}

// FaceEnrollment is the stored face template of an employee.
// Only the embedding is persisted — never raw face photos
// (on-device architecture, Milestone 2).
type FaceEnrollment struct {
	ID           int64     `json:"id"`
	EmployeeID   int64     `json:"employee_id"`
	Embedding    []float32 `json:"-"` // binary blob in DB; never exposed via JSON
	Dimension    int       `json:"dimension"`
	QualityScore float64   `json:"quality_score"`
	EnrolledAt   time.Time `json:"enrolled_at"`
}

// SpoofAttempt is one rejected liveness check (Milestone 3 anti-spoofing
// audit log). Repeated attempts within the lockout window temporarily
// lock face verification for the employee.
type SpoofAttempt struct {
	ID            int64     `json:"id"`
	EmployeeID    int64     `json:"employee_id"`
	LivenessScore float64   `json:"liveness_score"`
	Reason        string    `json:"reason"`
	DeviceID      *string   `json:"device_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
