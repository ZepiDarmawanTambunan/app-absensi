//go:build integration

// Integration tests for the repository layer. They run against a real MySQL
// database and are excluded from the default `go test ./...` run.
//
// Run them with a dedicated TEST database (never shared/prod):
//
//	createdb: CREATE DATABASE absensi_test;
//	INTEGRATION_DB_DSN="root@tcp(127.0.0.1:3306)/absensi_test?parseTime=true&multiStatements=true" \
//	  go test -tags=integration ./internal/repository/
package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/model"
)

// setupTestDB migrates a scratch schema up, truncates all tables and
// registers a down-migration cleanup. It skips when INTEGRATION_DB_DSN is unset.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("INTEGRATION_DB_DSN")
	if dsn == "" {
		t.Skip("INTEGRATION_DB_DSN not set; skipping integration test")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("cannot reach test DB (is MySQL running?): %v", err)
	}
	up, err := os.ReadFile("../../migrations/000001_init.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	if _, err := db.Exec(string(up)); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	// Truncate in dependency order for a clean slate.
	for _, tbl := range []string{"attendance_logs", "devices", "employees", "users"} {
		if _, err := db.Exec("DELETE FROM " + tbl); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
	t.Cleanup(func() {
		down, err := os.ReadFile("../../migrations/000001_init.down.sql")
		if err == nil {
			_, _ = db.Exec(string(down))
		}
		_ = db.Close()
	})
	return db
}

func TestUserRepository_Integration(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := NewUserRepository(db)

	u := &model.User{Name: "Admin", Email: "admin@example.com", PasswordHash: "hashed", Role: "admin"}
	if err := repo.Create(ctx, db, u); err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expected generated id")
	}
	got, err := repo.FindByEmail(ctx, db, "admin@example.com")
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if got.Name != "Admin" {
		t.Fatalf("unexpected user: %+v", got)
	}
	if _, err := repo.FindByEmail(ctx, db, "missing@example.com"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestEmployeeRepository_Integration(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := NewEmployeeRepository(db)

	e := &model.Employee{EmployeeNo: "EMP001", Name: "Budi", Department: "IT", IsActive: true}
	if err := Transact(ctx, db, func(tx DBTX) error { return repo.Create(ctx, tx, e) }); err != nil {
		t.Fatalf("create: %v", err)
	}

	dup := &model.Employee{EmployeeNo: "EMP001", Name: "Budi 2", IsActive: true}
	if err := Transact(ctx, db, func(tx DBTX) error { return repo.Create(ctx, tx, dup) }); err == nil {
		t.Fatal("expected duplicate employee_no to fail")
	}

	got, err := repo.FindByEmployeeNo(ctx, db, "EMP001")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.Name != "Budi" {
		t.Fatalf("unexpected employee: %+v", got)
	}

	list, err := repo.List(ctx, db, false, 20, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 employee, got %d (%v)", len(list), err)
	}

	if err := Transact(ctx, db, func(tx DBTX) error { return repo.SoftDelete(ctx, tx, e.ID) }); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if _, err := repo.FindByID(ctx, db, e.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after soft delete, got %v", err)
	}
}

func TestAttendanceRepository_FlowIntegration(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	empRepo := NewEmployeeRepository(db)
	attRepo := NewAttendanceRepository(db)

	e := &model.Employee{EmployeeNo: "EMP001", Name: "Budi", IsActive: true}
	if err := Transact(ctx, db, func(tx DBTX) error { return empRepo.Create(ctx, tx, e) }); err != nil {
		t.Fatalf("create employee: %v", err)
	}

	if _, err := attRepo.FindOpenCheckIn(ctx, db, e.ID); err != ErrNotFound {
		t.Fatalf("expected no open check-in, got %v", err)
	}

	err := Transact(ctx, db, func(tx DBTX) error {
		in := &model.AttendanceLog{EmployeeID: e.ID, Type: model.AttendanceCheckIn, Method: model.MethodManual, Verified: true, RecordedAt: time.Now().Add(-time.Hour)}
		if err := attRepo.CreateLog(ctx, tx, in); err != nil {
			return err
		}
		empID := e.ID
		return attRepo.UpsertDevice(ctx, tx, &model.Device{DeviceID: "dev-1", EmployeeID: &empID, Platform: "android"})
	})
	if err != nil {
		t.Fatalf("check-in tx: %v", err)
	}

	open, err := attRepo.FindOpenCheckIn(ctx, db, e.ID)
	if err != nil {
		t.Fatalf("expected open check-in: %v", err)
	}
	if open.Method != model.MethodManual {
		t.Fatalf("unexpected open log: %+v", open)
	}

	err = Transact(ctx, db, func(tx DBTX) error {
		return attRepo.CreateLog(ctx, tx, &model.AttendanceLog{
			EmployeeID: e.ID, Type: model.AttendanceCheckOut, Method: model.MethodManual, Verified: true, RecordedAt: time.Now(),
		})
	})
	if err != nil {
		t.Fatalf("check-out tx: %v", err)
	}

	if _, err := attRepo.FindOpenCheckIn(ctx, db, e.ID); err != ErrNotFound {
		t.Fatalf("expected check-in to be closed, got %v", err)
	}

	logs, err := attRepo.ListByEmployee(ctx, db, e.ID, time.Now().AddDate(0, 0, -1), time.Now().Add(time.Hour), 20, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	if logs[0].Type != model.AttendanceCheckOut {
		t.Fatal("expected newest-first ordering")
	}
}

func TestTransact_RollbackIntegration(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := NewUserRepository(db)

	err := Transact(ctx, db, func(tx DBTX) error {
		if err := repo.Create(ctx, tx, &model.User{Name: "Tmp", Email: "tmp@example.com", PasswordHash: "x", Role: "admin"}); err != nil {
			return err
		}
		return sql.ErrTxDone // force rollback
	})
	if err == nil {
		t.Fatal("expected forced error")
	}
	if _, err := repo.FindByEmail(ctx, db, "tmp@example.com"); err != ErrNotFound {
		t.Fatalf("expected rollback to discard the row, got %v", err)
	}
}
