package collector

import (
	"github.com/homagames/incident-io-exporter/internal/client"
	"github.com/prometheus/client_golang/prometheus"
)

func collectAlerts(ch chan<- prometheus.Metric, desc *prometheus.Desc, alerts []client.Alert) {
	type key struct {
		status, team string
	}
	counts := make(map[key]float64)
	for _, a := range alerts {
		counts[key{status: a.Status, team: a.Team()}]++
	}
	for k, count := range counts {
		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			count,
			k.status, k.team,
		)
	}
}
