// Package config loads and validates environment-based configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// ConfigError represents a configuration error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error: field %q: %s", e.Field, e.Message)
}

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DBDSN        string
	GoogleAPIKey string
	Port         int

	// JWT authentication settings.
	JWTSecret       string // Required for auth endpoints; signing key for HS256.
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// File uploads.
	UploadDir string // Directory for uploaded images; defaults to "./uploads/images".
}

// Load reads and validates required environment variables.
// Returns a ConfigError for any missing or invalid value.
func Load() (*Config, error) {
	cfg := &Config{}

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		return nil, &ConfigError{Field: "DB_DSN", Message: "required but not set"}
	}
	cfg.DBDSN = dbDSN

	cfg.GoogleAPIKey = os.Getenv("GOOGLE_API_KEY")
	// Not strictly required for bootstrap; warn but don't fail.

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	// Not required at startup; auth endpoints will fail gracefully if unset.

	cfg.AccessTokenTTL = parseDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute)
	cfg.RefreshTokenTTL = parseDurationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour)

	cfg.UploadDir = os.Getenv("UPLOAD_DIR")
	if cfg.UploadDir == "" {
		cfg.UploadDir = "./uploads/images"
	}

	portStr := os.Getenv("PORT")
	if portStr == "" {
		cfg.Port = 8080
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, &ConfigError{Field: "PORT", Message: "must be a valid integer"}
		}
		if port < 1 || port > 65535 {
			return nil, &ConfigError{Field: "PORT", Message: "must be between 1 and 65535"}
		}
		cfg.Port = port
	}

	return cfg, nil
}

// Validate re-checks required fields on an already-constructed Config.
func (c *Config) Validate() error {
	var errs []error
	if c.DBDSN == "" {
		errs = append(errs, &ConfigError{Field: "DB_DSN", Message: "cannot be empty"})
	}
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, &ConfigError{Field: "PORT", Message: "must be between 1 and 65535"})
	}
	return errors.Join(errs...)
}

// parseDurationEnv reads a duration from an environment variable.
// Falls back to defaultVal if the variable is unset or unparseable.
// Accepts Go duration strings like "15m", "24h", "168h".
func parseDurationEnv(key string, defaultVal time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultVal
	}
	return d
}
