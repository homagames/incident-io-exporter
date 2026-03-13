package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_RequiresAPIKey(t *testing.T) {
	os.Unsetenv("INCIDENT_IO_API_KEY")
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "INCIDENT_IO_API_KEY is required")
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("INCIDENT_IO_API_KEY", "test-key")
	os.Unsetenv("LISTEN_ADDRESS")
	os.Unsetenv("METRICS_PATH")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SCRAPE_INTERVAL")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "test-key", cfg.APIKey)
	assert.Equal(t, ":9090", cfg.ListenAddress)
	assert.Equal(t, "/metrics", cfg.MetricsPath)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 1*time.Minute, cfg.ScrapeInterval)
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("INCIDENT_IO_API_KEY", "my-key")
	t.Setenv("LISTEN_ADDRESS", ":8080")
	t.Setenv("METRICS_PATH", "/prom")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SCRAPE_INTERVAL", "30s")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "my-key", cfg.APIKey)
	assert.Equal(t, ":8080", cfg.ListenAddress)
	assert.Equal(t, "/prom", cfg.MetricsPath)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 30*time.Second, cfg.ScrapeInterval)
}

func TestLoad_InvalidScrapeInterval(t *testing.T) {
	t.Setenv("INCIDENT_IO_API_KEY", "key")
	t.Setenv("SCRAPE_INTERVAL", "not-a-duration")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid SCRAPE_INTERVAL")
}

func TestLoad_ScrapeIntervalTooLow(t *testing.T) {
	t.Setenv("INCIDENT_IO_API_KEY", "key")
	t.Setenv("SCRAPE_INTERVAL", "5s")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 10s")
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_VAR", "value")
	assert.Equal(t, "value", getEnvOrDefault("TEST_VAR", "default"))

	os.Unsetenv("TEST_VAR_MISSING")
	assert.Equal(t, "default", getEnvOrDefault("TEST_VAR_MISSING", "default"))
}
