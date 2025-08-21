package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"spaudit/database"
	"spaudit/logging"
)

// AppConfig holds application-wide system configuration.
// This is infrastructure configuration, not user audit preferences.
type AppConfig struct {
	HTTPAddr    string
	HTTPLogPath string
	Database    *database.Config
	Logging     *logging.Config
}

// LoadAppConfigFromEnv loads complete application configuration from environment variables.
func LoadAppConfigFromEnv() *AppConfig {
	return &AppConfig{
		HTTPAddr:    getEnvWithDefault("HTTP_ADDR", ":8080"),
		HTTPLogPath: getEnvWithDefault("HTTP_LOG_PATH", ""),
		Database:    LoadDatabaseConfigFromEnv(),
		Logging:     LoadLoggingConfigFromEnv(),
	}
}

// LoadDatabaseConfigFromEnv loads database configuration from environment variables.
func LoadDatabaseConfigFromEnv() *database.Config {
	return &database.Config{
		Path:              getEnvWithDefault("DB_PATH", "./spaudit.db"),
		MaxOpenConns:      getEnvIntWithDefault("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:      getEnvIntWithDefault("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime:   getEnvDurationWithDefault("DB_CONN_MAX_LIFETIME", time.Hour),
		ConnMaxIdleTime:   getEnvDurationWithDefault("DB_CONN_MAX_IDLE_TIME", 15*time.Minute),
		BusyTimeoutMs:     getEnvIntWithDefault("DB_BUSY_TIMEOUT_MS", 5000),
		EnableForeignKeys: getEnvBoolWithDefault("DB_ENABLE_FOREIGN_KEYS", true),
		EnableWAL:         getEnvBoolWithDefault("DB_ENABLE_WAL", true),
		StrictMode:        getEnvBoolWithDefault("DB_STRICT_MODE", true),
	}
}

// LoadLoggingConfigFromEnv loads logging configuration from environment variables.
func LoadLoggingConfigFromEnv() *logging.Config {
	return &logging.Config{
		Level:  getEnvWithDefault("LOG_LEVEL", "info"),
		Format: getEnvWithDefault("LOG_FORMAT", "json"),
		Output: getEnvWithDefault("LOG_OUTPUT", "stdout"),
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseBool(v string, def bool) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

// Helper functions for environment variable parsing.
func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBoolWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return parseBool(value, defaultValue)
	}
	return defaultValue
}

func getEnvDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
