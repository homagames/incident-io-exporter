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
}

// Collector implements prometheus.Collector for incident.io metrics.
// It caches API results and only refreshes after the configured interval.
type Collector struct {
	client         APIClient
	scrapeInterval time.Duration

	incidentsTotal   *prometheus.Desc
	incidentDuration *prometheus.Desc
	alertsTotal      *prometheus.Desc
	severities       *prometheus.Desc
	scrapeDuration   *prometheus.Desc
	scrapeErrors     *prometheus.Desc
	buildInfo        *prometheus.Desc

	mu          sync.Mutex
	lastScrape  time.Time
	cachedData  *scrapeResult
	version     string
}

type scrapeResult struct {
	incidents []client.Incident
	alerts    []client.Alert
	sevs      []client.Severity
	errors    float64
	duration  time.Duration
}

// New creates a new Collector.
func New(apiClient APIClient, scrapeInterval time.Duration, version string) *Collector {
	return &Collector{
		client:         apiClient,
		scrapeInterval: scrapeInterval,
		version:        version,
		incidentsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "incidents", "total"),
			"Total number of incidents by status, severity, type, mode, visibility, and team.",
			[]string{"status", "severity", "type", "mode", "visibility", "team"},
			nil,
		),
		incidentDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "incidents", "duration_seconds"),
			"Duration metric values for incidents.",
			[]string{"severity", "metric"},
			nil,
		),
		alertsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "alerts", "total"),
			"Total number of alerts by status and team.",
			[]string{"status", "team"},
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
		buildInfo: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "build_info"),
			"Build information for the exporter.",
			[]string{"version"},
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
	ch <- c.buildInfo
}

// Collect fetches data from incident.io and sends metrics to the channel.
// Results are cached for the configured scrape interval.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	if c.cachedData != nil && time.Since(c.lastScrape) < c.scrapeInterval {
		data := c.cachedData
		c.mu.Unlock()
		c.emitMetrics(ch, data)
		return
	}
	c.mu.Unlock()

	data := c.fetchAll()

	c.mu.Lock()
	c.cachedData = data
	c.lastScrape = time.Now()
	c.mu.Unlock()

	c.emitMetrics(ch, data)
}

func (c *Collector) fetchAll() *scrapeResult {
	start := time.Now()
	var scrapeErrors float64

	var (
		incidents []client.Incident
		alerts    []client.Alert
		sevs      []client.Severity
		wg        sync.WaitGroup
		mu        sync.Mutex
		incidentErr, alertErr, sevErr error
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
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
	}
	if alertErr != nil {
		slog.Error("failed to list alerts", "error", alertErr)
		scrapeErrors++
	}
	if sevErr != nil {
		slog.Error("failed to list severities", "error", sevErr)
		scrapeErrors++
	}

	return &scrapeResult{
		incidents: incidents,
		alerts:    alerts,
		sevs:      sevs,
		errors:    scrapeErrors,
		duration:  time.Since(start),
	}
}

func (c *Collector) emitMetrics(ch chan<- prometheus.Metric, data *scrapeResult) {
	if data.incidents != nil {
		collectIncidents(ch, c.incidentsTotal, c.incidentDuration, data.incidents)
	}
	if data.alerts != nil {
		collectAlerts(ch, c.alertsTotal, data.alerts)
	}
	if data.sevs != nil {
		collectSeverities(ch, c.severities, data.sevs)
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeDuration, prometheus.GaugeValue, data.duration.Seconds())
	ch <- prometheus.MustNewConstMetric(c.scrapeErrors, prometheus.GaugeValue, data.errors)
	ch <- prometheus.MustNewConstMetric(c.buildInfo, prometheus.GaugeValue, 1, c.version)
}
