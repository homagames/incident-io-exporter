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

	for _, inc := range incidents {
		severity := ""
		if inc.Severity != nil {
			severity = inc.Severity.Name
		}
		incType := ""
		if inc.Type != nil {
			incType = inc.Type.Name
		}

		k := key{
			status:       inc.Status.Name,
			severity:     severity,
			incidentType: incType,
			mode:         inc.Mode,
			visibility:   inc.Visibility,
			team:         inc.Team(),
		}
		counts[k]++

		// Emit duration metrics per incident
		for _, d := range inc.Duration {
			ch <- prometheus.MustNewConstMetric(
				durationDesc,
				prometheus.GaugeValue,
				float64(d.ValueSeconds),
				inc.Reference,
				severity,
				d.Metric.Name,
			)
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
