package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration.
type Config struct {
	// Server configuration
	Host string
	Port int

	// Logging configuration
	LogLevel string

	// Application configuration
	Environment string
}

// Load loads the configuration from environment variables.
func Load() *Config {
	config := &Config{
		Host:        getEnv("HOST", "localhost"),
		Port:        getEnvAsInt("PORT", 8080),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	return config
}

// getEnv gets an environment variable with a default value.
func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

// getEnvAsInt gets an environment variable as integer with a default value.
func getEnvAsInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}