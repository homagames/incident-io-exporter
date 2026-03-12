package collector

import (
	"github.com/homagames/incident-io-exporter/internal/client"
	"github.com/prometheus/client_golang/prometheus"
)

func collectAlerts(ch chan<- prometheus.Metric, desc *prometheus.Desc, alerts []client.Alert) {
	counts := make(map[string]float64)
	for _, a := range alerts {
		counts[a.Status]++
	}
	for status, count := range counts {
		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			count,
			status,
		)
	}
}
