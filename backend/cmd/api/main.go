// Command api runs the attendance REST API service.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/config"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/handler"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/middleware"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/repository"
	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	db, err := repository.Open(cfg.DBDSN)
	if err != nil {
		slog.Error("database error", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	transact := service.DBTransactor(db)

	// Wire dependencies: repository -> service -> handler.
	userRepo := repository.NewUserRepository(db)
	employeeRepo := repository.NewEmployeeRepository(db)
	attendanceRepo := repository.NewAttendanceRepository(db)

	authSvc := service.NewAuthService(db, userRepo, transact, cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)
	employeeSvc := service.NewEmployeeService(db, employeeRepo, transact)
	attendanceSvc := service.NewAttendanceService(db, attendanceRepo, transact)

	authH := handler.NewAuthHandler(authSvc)
	employeeH := handler.NewEmployeeHandler(employeeSvc)
	attendanceH := handler.NewAttendanceHandler(attendanceSvc)

	faceRepo := repository.NewFaceEnrollmentRepository(db)
	spoofRepo := repository.NewSpoofAttemptRepository(db)
	faceSvc := service.NewFaceService(db, employeeRepo, faceRepo, spoofRepo, transact,
		cfg.FaceMatchThreshold,
		service.LivenessOptions{
			Threshold:      cfg.FaceLivenessThreshold,
			Required:       cfg.FaceLivenessRequired,
			MaxFailures:    cfg.MaxLivenessFailures,
			LockoutMinutes: cfg.LivenessLockoutMins,
			Challenges:     service.NewChallengeStore(service.ChallengeTTL),
		})
	faceH := handler.NewFaceHandler(faceSvc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"status":"ok"}}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Post("/employees", employeeH.Create)
			r.Get("/employees", employeeH.List)
			r.Post("/attendance/check-in", attendanceH.CheckIn)
			r.Post("/attendance/check-out", attendanceH.CheckOut)
			r.Get("/attendance", attendanceH.History)
			r.Post("/face/enroll", faceH.Enroll)
			r.Post("/face/verify", faceH.Verify)
			r.Get("/face/challenge", faceH.Challenge)
		})
	})

	slog.Info("listening", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
