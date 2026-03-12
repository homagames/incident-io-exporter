package collector

import (
	"errors"
	"testing"
	"time"

	"github.com/homagames/incident-io-exporter/internal/client"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements APIClient for testing.
type mockClient struct {
	incidents  []client.Incident
	alerts     []client.Alert
	severities []client.Severity
	incErr     error
	alertErr   error
	sevErr     error
}

func (m *mockClient) ListIncidents() ([]client.Incident, error) {
	return m.incidents, m.incErr
}

func (m *mockClient) ListAlerts() ([]client.Alert, error) {
	return m.alerts, m.alertErr
}

func (m *mockClient) ListSeverities() ([]client.Severity, error) {
	return m.severities, m.sevErr
}

func newTestCollector(mc *mockClient) *Collector {
	return New(mc, 1*time.Minute, "test")
}

func collectMetrics(t *testing.T, c *Collector) []*dto.MetricFamily {
	t.Helper()
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)
	families, err := reg.Gather()
	require.NoError(t, err)
	return families
}

func findMetricFamily(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func TestCollectorDescribe(t *testing.T) {
	c := newTestCollector(&mockClient{})
	ch := make(chan *prometheus.Desc, 10)
	c.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for d := range ch {
		descs = append(descs, d)
	}
	assert.Len(t, descs, 7) // 6 original + buildInfo
}

func TestCollect_Incidents(t *testing.T) {
	mc := &mockClient{
		incidents: []client.Incident{
			{
				ID:   "1",
				Mode: "standard",
				Status: client.IncidentStatus{
					Name:     "Active",
					Category: "live",
				},
				Severity:   &client.Severity{Name: "Critical"},
				Visibility: "public",
			},
			{
				ID:   "2",
				Mode: "standard",
				Status: client.IncidentStatus{
					Name:     "Active",
					Category: "live",
				},
				Severity:   &client.Severity{Name: "Critical"},
				Visibility: "public",
			},
			{
				ID:   "3",
				Mode: "standard",
				Status: client.IncidentStatus{
					Name:     "Closed",
					Category: "closed",
				},
				Severity:   &client.Severity{Name: "Minor"},
				Visibility: "public",
			},
		},
		severities: []client.Severity{},
	}

	c := newTestCollector(mc)
	families := collectMetrics(t, c)

	incTotal := findMetricFamily(families, "incidentio_incidents_total")
	require.NotNil(t, incTotal)
	assert.GreaterOrEqual(t, len(incTotal.GetMetric()), 2) // at least 2 label combos
}

func TestCollect_Alerts(t *testing.T) {
	mc := &mockClient{
		alerts: []client.Alert{
			{
				ID:     "a1",
				Status: "firing",
				Attributes: []client.AlertAttributeEntry{
					{Attribute: client.AlertAttribute{Name: "Team"}, Value: &client.AlertAttributeValue{Label: "Infra"}},
				},
			},
			{
				ID:     "a2",
				Status: "firing",
				Attributes: []client.AlertAttributeEntry{
					{Attribute: client.AlertAttribute{Name: "Team"}, Value: &client.AlertAttributeValue{Label: "Infra"}},
				},
			},
			{
				ID:     "a3",
				Status: "resolved",
				Attributes: []client.AlertAttributeEntry{
					{Attribute: client.AlertAttribute{Name: "Team"}, Value: &client.AlertAttributeValue{Label: "Platform"}},
				},
			},
		},
		severities: []client.Severity{},
	}

	c := newTestCollector(mc)
	families := collectMetrics(t, c)

	alertTotal := findMetricFamily(families, "incidentio_alerts_total")
	require.NotNil(t, alertTotal)

	metrics := alertTotal.GetMetric()
	assert.Len(t, metrics, 2) // firing/Infra + resolved/Platform

	for _, m := range metrics {
		labels := map[string]string{}
		for _, l := range m.GetLabel() {
			labels[l.GetName()] = l.GetValue()
		}
		if labels["status"] == "firing" {
			assert.Equal(t, "Infra", labels["team"])
			assert.Equal(t, float64(2), m.GetGauge().GetValue())
		}
		if labels["status"] == "resolved" {
			assert.Equal(t, "Platform", labels["team"])
			assert.Equal(t, float64(1), m.GetGauge().GetValue())
		}
	}
}

func TestCollect_Severities(t *testing.T) {
	mc := &mockClient{
		severities: []client.Severity{
			{Name: "Critical", Rank: 1},
			{Name: "Major", Rank: 2},
		},
	}

	c := newTestCollector(mc)
	families := collectMetrics(t, c)

	sevs := findMetricFamily(families, "incidentio_severities")
	require.NotNil(t, sevs)
	assert.Len(t, sevs.GetMetric(), 2)
}

func TestCollect_DurationMetrics(t *testing.T) {
	mc := &mockClient{
		incidents: []client.Incident{
			{
				ID:        "1",
				Reference: "INC-1",
				Mode:      "standard",
				Status:    client.IncidentStatus{Name: "Closed"},
				Severity:  &client.Severity{Name: "Critical"},
				Duration: []client.DurationValue{
					{
						Metric:       client.DurationMetric{Name: "time_to_resolve"},
						ValueSeconds: 3600,
					},
				},
			},
			{
				ID:        "2",
				Reference: "INC-2",
				Mode:      "standard",
				Status:    client.IncidentStatus{Name: "Closed"},
				Severity:  &client.Severity{Name: "Critical"},
				Duration: []client.DurationValue{
					{
						Metric:       client.DurationMetric{Name: "time_to_resolve"},
						ValueSeconds: 1800,
					},
				},
			},
		},
		severities: []client.Severity{},
	}

	c := newTestCollector(mc)
	families := collectMetrics(t, c)

	dur := findMetricFamily(families, "incidentio_incidents_duration_seconds")
	require.NotNil(t, dur)
	assert.Len(t, dur.GetMetric(), 1) // aggregated by severity+metric
	// Average of 3600 and 1800 = 2700
	assert.Equal(t, float64(2700), dur.GetMetric()[0].GetGauge().GetValue())
}

func TestCollect_ScrapeMetrics(t *testing.T) {
	mc := &mockClient{
		severities: []client.Severity{},
	}

	c := newTestCollector(mc)
	families := collectMetrics(t, c)

	sd := findMetricFamily(families, "incidentio_scrape_duration_seconds")
	require.NotNil(t, sd)

	se := findMetricFamily(families, "incidentio_scrape_errors_total")
	require.NotNil(t, se)
	assert.Equal(t, float64(0), se.GetMetric()[0].GetGauge().GetValue())
}

func TestCollect_BuildInfo(t *testing.T) {
	mc := &mockClient{
		severities: []client.Severity{},
	}

	c := New(mc, 1*time.Minute, "1.2.3")
	families := collectMetrics(t, c)

	bi := findMetricFamily(families, "incidentio_build_info")
	require.NotNil(t, bi)
	assert.Len(t, bi.GetMetric(), 1)

	labels := map[string]string{}
	for _, l := range bi.GetMetric()[0].GetLabel() {
		labels[l.GetName()] = l.GetValue()
	}
	assert.Equal(t, "1.2.3", labels["version"])
	assert.Equal(t, float64(1), bi.GetMetric()[0].GetGauge().GetValue())
}

func TestCollect_WithErrors(t *testing.T) {
	mc := &mockClient{
		incErr:   errors.New("incidents failed"),
		alertErr: errors.New("alerts failed"),
		sevErr:   errors.New("severities failed"),
	}

	c := newTestCollector(mc)
	families := collectMetrics(t, c)

	se := findMetricFamily(families, "incidentio_scrape_errors_total")
	require.NotNil(t, se)
	assert.Equal(t, float64(3), se.GetMetric()[0].GetGauge().GetValue())
}

func TestCollect_IncidentsWithNilSeverityAndType(t *testing.T) {
	mc := &mockClient{
		incidents: []client.Incident{
			{
				ID:         "1",
				Mode:       "standard",
				Visibility: "public",
				Status:     client.IncidentStatus{Name: "Active"},
				Severity:   nil,
				Type:       nil,
			},
		},
		severities: []client.Severity{},
	}

	c := newTestCollector(mc)
	families := collectMetrics(t, c)

	incTotal := findMetricFamily(families, "incidentio_incidents_total")
	require.NotNil(t, incTotal)
	assert.Len(t, incTotal.GetMetric(), 1)
}

func TestCollect_CachingBehavior(t *testing.T) {
	callCount := 0
	mc := &mockClient{
		severities: []client.Severity{},
	}

	// Wrap to count calls
	c := New(mc, 1*time.Minute, "test")

	// First collect should fetch
	reg := prometheus.NewRegistry()
	reg.MustRegister(c)
	_, err := reg.Gather()
	require.NoError(t, err)

	// Record the cached data
	c.mu.Lock()
	firstScrape := c.lastScrape
	c.mu.Unlock()
	assert.False(t, firstScrape.IsZero())

	// Second collect within interval should use cache (lastScrape unchanged)
	_, err = reg.Gather()
	require.NoError(t, err)

	c.mu.Lock()
	secondScrape := c.lastScrape
	c.mu.Unlock()
	assert.Equal(t, firstScrape, secondScrape)

	_ = callCount
}
