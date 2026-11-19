package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	labelBlock   = "block"
	labelFormat  = "format"
	labelQueue   = "queue"
	labelOutcome = "outcome"
	labelFailure = "failure"

	blockTerrain = "terrain"
	blockClimate = "climate"

	failureRetry     = "retry"
	failureAbandoned = "abandoned"
)

var messageAgeBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300}

type BacklogSource interface {
	Backlog(requestContext context.Context) (domain.OutboxBacklog, error)
}

type ReportCountSource interface {
	Counts(requestContext context.Context) (domain.ReportCounts, error)
}

type Stats struct {
	Backlog BacklogSource
	Reports ReportCountSource
	Timeout time.Duration
}

type Service struct {
	registry          *prometheus.Registry
	requestDuration   *prometheus.HistogramVec
	clientDuration    *prometheus.HistogramVec
	reports           prometheus.Counter
	unavailableBlocks *prometheus.CounterVec
	downloads         *prometheus.CounterVec
	outboxPublished   prometheus.Counter
	outboxFailures    *prometheus.CounterVec
	deliveries        *prometheus.CounterVec
	messageAge        *prometheus.HistogramVec
	now               func() time.Time
}

func New(serviceName string, stats Stats) *Service {
	serviceLabel := prometheus.Labels{labelService: serviceName}
	counter := func(name, help string, labels ...string) *prometheus.CounterVec {
		return prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: name, Help: help, ConstLabels: serviceLabel},
			labels,
		)
	}

	service := &Service{
		registry:        prometheus.NewRegistry(),
		requestDuration: newRequestDuration(serviceLabel),
		clientDuration:  newClientDuration(serviceLabel),
		reports: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "reports_created_total", Help: "Reports created.", ConstLabels: serviceLabel,
		}),
		unavailableBlocks: counter(
			"report_blocks_unavailable_total",
			"Reports created while the terrain or climate source did not answer.",
			labelBlock,
		),
		downloads: counter("report_downloads_total", "Report files built for download, by format.", labelFormat),
		outboxPublished: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "outbox_published_total", Help: "Outbox messages confirmed by the broker.", ConstLabels: serviceLabel,
		}),
		outboxFailures: counter(
			"outbox_publish_errors_total",
			"Failed outbox publications: retried later or abandoned after the maximum attempts.",
			labelFailure,
		),
		deliveries: counter("messages_processed_total", "Consumed messages by outcome.", labelQueue, labelOutcome),
		messageAge: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "message_age_seconds",
			Help:        "Time from writing the event to the outbox until a consumer processed it.",
			Buckets:     messageAgeBuckets,
			ConstLabels: serviceLabel,
		}, []string{labelQueue}),
		now: time.Now,
	}

	service.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		newStatsCollector(stats, serviceLabel),
		service.requestDuration,
		service.clientDuration,
		service.reports,
		service.unavailableBlocks,
		service.downloads,
		service.outboxPublished,
		service.outboxFailures,
		service.deliveries,
		service.messageAge,
	)

	return service
}

func (service *Service) Handler() http.Handler {
	return promhttp.HandlerFor(service.registry, promhttp.HandlerOpts{Registry: service.registry})
}

func (service *Service) ObserveRequests(next http.Handler) http.Handler {
	return observeRequests(service.requestDuration, next)
}

func (service *Service) Transport(target string, base http.RoundTripper) http.RoundTripper {
	return observedTransport{base: base, target: target, duration: service.clientDuration}
}

func (service *Service) ReportCreated(terrain, climate domain.DataStatus) {
	service.reports.Inc()

	if terrain != domain.DataAvailable {
		service.unavailableBlocks.WithLabelValues(blockTerrain).Inc()
	}

	if climate != domain.DataAvailable {
		service.unavailableBlocks.WithLabelValues(blockClimate).Inc()
	}
}

func (service *Service) ReportDownloaded(extension string) {
	service.downloads.WithLabelValues(extension).Inc()
}

func (service *Service) OutboxPublished(count int) {
	service.outboxPublished.Add(float64(count))
}

func (service *Service) OutboxPublishFailed(abandoned bool) {
	failure := failureRetry
	if abandoned {
		failure = failureAbandoned
	}

	service.outboxFailures.WithLabelValues(failure).Inc()
}

func (service *Service) DeliveryProcessed(queue, outcome string, createdAt time.Time) {
	service.deliveries.WithLabelValues(queue, outcome).Inc()

	if !createdAt.IsZero() {
		service.messageAge.WithLabelValues(queue).Observe(max(0, service.now().Sub(createdAt).Seconds()))
	}
}
