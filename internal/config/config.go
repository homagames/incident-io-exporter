package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all configuration for the exporter.
type Config struct {
	APIKey         string
	ListenAddress  string
	MetricsPath    string
	LogLevel       string
	ScrapeInterval time.Duration
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	apiKey := os.Getenv("INCIDENT_IO_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("INCIDENT_IO_API_KEY is required")
	}

	scrapeInterval, err := time.ParseDuration(getEnvOrDefault("SCRAPE_INTERVAL", "1m"))
	if err != nil {
		return nil, fmt.Errorf("invalid SCRAPE_INTERVAL: %w", err)
	}
	if scrapeInterval < 10*time.Second {
		return nil, fmt.Errorf("SCRAPE_INTERVAL must be at least 10s")
	}

	return &Config{
		APIKey:         apiKey,
		ListenAddress:  getEnvOrDefault("LISTEN_ADDRESS", ":9090"),
		MetricsPath:    getEnvOrDefault("METRICS_PATH", "/metrics"),
		LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
		ScrapeInterval: scrapeInterval,
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
