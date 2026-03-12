package collector

import (
	"strconv"

	"github.com/homagames/incident-io-exporter/internal/client"
	"github.com/prometheus/client_golang/prometheus"
)

func collectIncidents(ch chan<- prometheus.Metric, totalDesc, durationDesc *prometheus.Desc, incidents []client.Incident) {
	// Count incidents by label combination
	type key struct {
		status, severity, incidentType, mode, visibility, team string
	}
	counts := make(map[key]float64)

	// Aggregate duration metrics by severity + metric name (average)
	type durKey struct {
		severity, metric string
	}
	durSums := make(map[durKey]float64)
	durCounts := make(map[durKey]float64)

	for _, inc := range incidents {
		severity := ""
		if inc.Severity != nil {
			severity = inc.Severity.Name
		}
		incType := ""
		if inc.Type != nil {
			incType = inc.Type.Name
		}

		for _, team := range inc.Teams() {
			k := key{
				status:       inc.Status.Name,
				severity:     severity,
				incidentType: incType,
				mode:         inc.Mode,
				visibility:   inc.Visibility,
				team:         team,
			}
			counts[k]++
		}

		for _, d := range inc.Duration {
			dk := durKey{severity: severity, metric: d.Metric.Name}
			durSums[dk] += float64(d.ValueSeconds)
			durCounts[dk]++
		}
	}

	for k, v := range counts {
		ch <- prometheus.MustNewConstMetric(
			totalDesc,
			prometheus.GaugeValue,
			v,
			k.status, k.severity, k.incidentType, k.mode, k.visibility, k.team,
		)
	}

	for dk, sum := range durSums {
		ch <- prometheus.MustNewConstMetric(
			durationDesc,
			prometheus.GaugeValue,
			sum/durCounts[dk],
			dk.severity, dk.metric,
		)
	}
}

func collectSeverities(ch chan<- prometheus.Metric, desc *prometheus.Desc, severities []client.Severity) {
	for _, s := range severities {
		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			1,
			s.Name,
			strconv.FormatInt(s.Rank, 10),
		)
	}
}
