// internal/config/config.go
package config

import (
	"os"
)

// Config holds all application configuration values.
type Config struct {
	ServerAddr string // e.g. ":8080"
	APIKey     string // API key required for /order requests
	CouponDir  string // directory containing coupon .gz files
}

// Load builds a Config struct using environment variables with fallbacks.
func Load() Config {
	cfg := Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8080"),
		APIKey:     getEnv("API_KEY", "apitest"),
		CouponDir:  getEnv("COUPON_DIR", "./data"),
	}
	return cfg
}

// helper: returns env var value if set, otherwise fallback default.
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
