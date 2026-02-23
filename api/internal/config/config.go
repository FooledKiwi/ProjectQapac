package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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
