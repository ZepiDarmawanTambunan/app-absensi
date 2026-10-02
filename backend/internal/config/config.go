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
	Port               string
	DBDSN              string
	JWTSecret          string
	AccessTTL          time.Duration
	RefreshTTL         time.Duration
	FaceMatchThreshold float64
}

// Load reads configuration from the environment and applies defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		DBDSN:              getEnv("DB_DSN", "absensi:absensi@tcp(127.0.0.1:3306)/absensi_db?parseTime=true"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		AccessTTL:          getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:         getDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		FaceMatchThreshold: getFloatEnv("FACE_MATCH_THRESHOLD", service.DefaultMatchThreshold),
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
