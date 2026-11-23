package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	statePending   = "pending"
	stateConfirmed = "confirmed"
	stateCancelled = "cancelled"
	stateDeleted   = "deleted"
)

type statsCollector struct {
	stats         Stats
	pending       *prometheus.Desc
	failed        *prometheus.Desc
	oldestAge     *prometheus.Desc
	plots         *prometheus.Desc
	scrapeFailure *prometheus.Desc
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
		plots: prometheus.NewDesc(
			"plots_stored", "Plots in plot_db by state.", []string{labelState}, serviceLabel,
		),
		scrapeFailure: prometheus.NewDesc(
			"database_stats_up",
			"1 when the outbox and plot statistics could be read on this scrape.",
			nil,
			serviceLabel,
		),
	}
}

func (collector *statsCollector) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- collector.pending

	descriptions <- collector.failed

	descriptions <- collector.oldestAge

	descriptions <- collector.plots

	descriptions <- collector.scrapeFailure
}

func (collector *statsCollector) Collect(metrics chan<- prometheus.Metric) {
	statsContext, cancel := context.WithTimeout(context.Background(), collector.stats.Timeout)
	defer cancel()

	backlog, backlogErr := collector.stats.Backlog.Backlog(statsContext)
	counts, countsErr := collector.stats.Plots.Counts(statsContext)

	up := 1.0
	if backlogErr != nil || countsErr != nil {
		up = 0
	}

	metrics <- prometheus.MustNewConstMetric(collector.scrapeFailure, prometheus.GaugeValue, up)

	if backlogErr == nil {
		metrics <- prometheus.MustNewConstMetric(collector.pending, prometheus.GaugeValue, float64(backlog.Pending))

		metrics <- prometheus.MustNewConstMetric(collector.failed, prometheus.GaugeValue, float64(backlog.Failed))

		metrics <- prometheus.MustNewConstMetric(collector.oldestAge, prometheus.GaugeValue, backlog.OldestAge.Seconds())
	}

	if countsErr == nil {
		for state, count := range map[string]int64{
			statePending:   counts.Pending,
			stateConfirmed: counts.Confirmed,
			stateCancelled: counts.Cancelled,
			stateDeleted:   counts.Deleted,
		} {
			metrics <- prometheus.MustNewConstMetric(collector.plots, prometheus.GaugeValue, float64(count), state)
		}
	}
}
