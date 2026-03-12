package collector

import (
	"log/slog"
	"sync"
	"time"

	"github.com/homagames/incident-io-exporter/internal/client"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "incidentio"

// APIClient defines the interface the collector needs from the incident.io client.
type APIClient interface {
	ListIncidents() ([]client.Incident, error)
	ListAlerts() ([]client.Alert, error)
	ListSeverities() ([]client.Severity, error)
	ListIncidentStatuses() ([]client.IncidentStatus, error)
}

// Collector implements prometheus.Collector for incident.io metrics.
type Collector struct {
	client APIClient

	incidentsTotal   *prometheus.Desc
	incidentDuration *prometheus.Desc
	alertsTotal      *prometheus.Desc
	severities       *prometheus.Desc
	scrapeDuration   *prometheus.Desc
	scrapeErrors     *prometheus.Desc
}

// New creates a new Collector.
func New(client APIClient) *Collector {
	return &Collector{
		client: client,
		incidentsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "incidents", "total"),
			"Total number of incidents by status, severity, type, mode, and visibility.",
			[]string{"status", "severity", "type", "mode", "visibility"},
			nil,
		),
		incidentDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "incidents", "duration_seconds"),
			"Duration metric values for incidents.",
			[]string{"incident", "severity", "metric"},
			nil,
		),
		alertsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "alerts", "total"),
			"Total number of alerts by status.",
			[]string{"status"},
			nil,
		),
		severities: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "severities"),
			"Configured severities with their rank.",
			[]string{"name", "rank"},
			nil,
		),
		scrapeDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "scrape", "duration_seconds"),
			"Time taken for the last scrape.",
			nil,
			nil,
		),
		scrapeErrors: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "scrape", "errors_total"),
			"Total number of scrape errors.",
			nil,
			nil,
		),
	}
}

// Describe sends the descriptors of each metric to the provided channel.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.incidentsTotal
	ch <- c.incidentDuration
	ch <- c.alertsTotal
	ch <- c.severities
	ch <- c.scrapeDuration
	ch <- c.scrapeErrors
}

// Collect fetches data from incident.io and sends metrics to the channel.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	var scrapeErrors float64

	var (
		incidents  []client.Incident
		alerts     []client.Alert
		sevs       []client.Severity
		wg         sync.WaitGroup
		mu         sync.Mutex
		incidentErr, alertErr, sevErr error
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		var err error
		i, err := c.client.ListIncidents()
		mu.Lock()
		incidents = i
		incidentErr = err
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		a, err := c.client.ListAlerts()
		mu.Lock()
		alerts = a
		alertErr = err
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		s, err := c.client.ListSeverities()
		mu.Lock()
		sevs = s
		sevErr = err
		mu.Unlock()
	}()
	wg.Wait()

	if incidentErr != nil {
		slog.Error("failed to list incidents", "error", incidentErr)
		scrapeErrors++
	} else {
		collectIncidents(ch, c.incidentsTotal, c.incidentDuration, incidents)
	}

	if alertErr != nil {
		slog.Error("failed to list alerts", "error", alertErr)
		scrapeErrors++
	} else {
		collectAlerts(ch, c.alertsTotal, alerts)
	}

	if sevErr != nil {
		slog.Error("failed to list severities", "error", sevErr)
		scrapeErrors++
	} else {
		collectSeverities(ch, c.severities, sevs)
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeDuration, prometheus.GaugeValue, time.Since(start).Seconds())
	ch <- prometheus.MustNewConstMetric(c.scrapeErrors, prometheus.GaugeValue, scrapeErrors)
}
