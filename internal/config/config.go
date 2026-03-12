package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the exporter.
type Config struct {
	APIKey        string
	ListenAddress string
	MetricsPath   string
	LogLevel      string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	apiKey := os.Getenv("INCIDENT_IO_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("INCIDENT_IO_API_KEY is required")
	}

	return &Config{
		APIKey:        apiKey,
		ListenAddress: getEnvOrDefault("LISTEN_ADDRESS", ":9090"),
		MetricsPath:   getEnvOrDefault("METRICS_PATH", "/metrics"),
		LogLevel:      getEnvOrDefault("LOG_LEVEL", "info"),
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
