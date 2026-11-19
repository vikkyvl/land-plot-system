package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelPlot = "plot"

	plotActive  = "active"
	plotDeleted = "deleted"
)

type statsCollector struct {
	stats     Stats
	pending   *prometheus.Desc
	failed    *prometheus.Desc
	oldestAge *prometheus.Desc
	reports   *prometheus.Desc
	up        *prometheus.Desc
}

func newStatsCollector(stats Stats, serviceLabel prometheus.Labels) *statsCollector {
	return &statsCollector{
		stats: stats,
		pending: prometheus.NewDesc(
			"outbox_pending", "Outbox rows not yet confirmed by the broker.", nil, serviceLabel,
		),
		failed: prometheus.NewDesc(
			"outbox_failed", "Outbox rows abandoned after the maximum attempts.", nil, serviceLabel,
		),
		oldestAge: prometheus.NewDesc(
			"outbox_oldest_age_seconds",
			"Age of the oldest pending outbox row, 0 when none is pending.",
			nil,
			serviceLabel,
		),
		reports: prometheus.NewDesc(
			"reports_stored",
			"Reports in report_db by whether their plot still exists.",
			[]string{labelPlot},
			serviceLabel,
		),
		up: prometheus.NewDesc(
			"database_stats_up",
			"1 when the outbox and report statistics could be read on this scrape.",
			nil,
			serviceLabel,
		),
	}
}

func (collector *statsCollector) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- collector.pending

	descriptions <- collector.failed

	descriptions <- collector.oldestAge

	descriptions <- collector.reports

	descriptions <- collector.up
}

func (collector *statsCollector) Collect(metrics chan<- prometheus.Metric) {
	statsContext, cancel := context.WithTimeout(context.Background(), collector.stats.Timeout)
	defer cancel()

	backlog, backlogErr := collector.stats.Backlog.Backlog(statsContext)
	counts, countsErr := collector.stats.Reports.Counts(statsContext)

	up := 1.0
	if backlogErr != nil || countsErr != nil {
		up = 0
	}

	metrics <- prometheus.MustNewConstMetric(collector.up, prometheus.GaugeValue, up)

	if backlogErr == nil {
		metrics <- prometheus.MustNewConstMetric(collector.pending, prometheus.GaugeValue, float64(backlog.Pending))

		metrics <- prometheus.MustNewConstMetric(collector.failed, prometheus.GaugeValue, float64(backlog.Failed))

		metrics <- prometheus.MustNewConstMetric(collector.oldestAge, prometheus.GaugeValue, backlog.OldestAge.Seconds())
	}

	if countsErr == nil {
		metrics <- prometheus.MustNewConstMetric(collector.reports, prometheus.GaugeValue, float64(counts.ActivePlots), plotActive)

		metrics <- prometheus.MustNewConstMetric(collector.reports, prometheus.GaugeValue, float64(counts.DeletedPlots), plotDeleted)
	}
}
