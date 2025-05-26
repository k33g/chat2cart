package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	OpenAIAPIKey string
	Port         string
	Environment  string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
		Port:         getEnv("PORT", "8080"),
		Environment:  getEnv("ENVIRONMENT", "development"),
	}
}

// getEnv gets an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetIntEnv gets an environment variable as integer with a fallback default value
func GetIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.OpenAIAPIKey == "" {
		return &ConfigError{
			Field:   "OPENAI_API_KEY",
			Message: "OpenAI API key is required. Please set the OPENAI_API_KEY environment variable.",
		}
	}
	return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
