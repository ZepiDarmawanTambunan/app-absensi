// Package config loads service configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ZepiDarmawanTambunan/absensi-face-recognition/backend/internal/service"
)

// Config holds all runtime configuration for the API service.
type Config struct {
	Port                  string
	DBDSN                 string
	JWTSecret             string
	AccessTTL             time.Duration
	RefreshTTL            time.Duration
	FaceMatchThreshold    float64
	FaceLivenessThreshold float64
	FaceLivenessRequired  bool
	MaxLivenessFailures   int
	LivenessLockoutMins   int
}

// Load reads configuration from the environment and applies defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                  getEnv("PORT", "8080"),
		DBDSN:                 getEnv("DB_DSN", "absensi:absensi@tcp(127.0.0.1:3306)/absensi_db?parseTime=true"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		AccessTTL:             getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:            getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		FaceMatchThreshold:    getFloatEnv("FACE_MATCH_THRESHOLD", service.DefaultMatchThreshold),
		FaceLivenessThreshold: getFloatEnv("FACE_LIVENESS_THRESHOLD", service.DefaultLivenessThreshold),
		FaceLivenessRequired:  getBoolEnv("FACE_LIVENESS_REQUIRED", true),
		MaxLivenessFailures:   getIntEnv("MAX_LIVENESS_FAILURES", service.DefaultMaxLivenessFailures),
		LivenessLockoutMins:   getIntEnv("LIVENESS_LOCKOUT_MINUTES", service.DefaultLivenessLockoutMinutes),
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDurationEnv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if secs, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return def
}

func getFloatEnv(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getBoolEnv(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getIntEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
