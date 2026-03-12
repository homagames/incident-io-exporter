package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/homagames/incident-io-exporter/internal/client"
	"github.com/homagames/incident-io-exporter/internal/collector"
	"github.com/homagames/incident-io-exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	setupLogging(cfg.LogLevel)

	apiClient := client.New(cfg.APIKey)
	coll := collector.New(apiClient)
	prometheus.MustRegister(coll)

	http.Handle(cfg.MetricsPath, promhttp.Handler())
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><h1>incident.io Exporter</h1><p><a href="` + cfg.MetricsPath + `">Metrics</a></p></body></html>`))
	})

	slog.Info("starting incident.io exporter", "address", cfg.ListenAddress, "metrics_path", cfg.MetricsPath)
	if err := http.ListenAndServe(cfg.ListenAddress, nil); err != nil {
		slog.Error("http server failed", "error", err)
		os.Exit(1)
	}
}

func setupLogging(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))
}
