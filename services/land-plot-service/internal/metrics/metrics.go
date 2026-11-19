package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const (
	labelChange  = "change"
	labelResult  = "result"
	labelReason  = "reason"
	labelKind    = "kind"
	labelQueue   = "queue"
	labelOutcome = "outcome"
	labelFailure = "failure"
	labelState   = "state"

	previewValid   = "valid"
	previewInvalid = "invalid"

	sagaConfirmed = "confirmed"
	sagaCancelled = "cancelled"
	sagaReverted  = "reverted"
	noReason      = "none"

	failureRetry     = "retry"
	failureAbandoned = "abandoned"
)

var (
	changeLabels = map[domain.PlotEventType]string{
		domain.PlotCreated: "created",
		domain.PlotUpdated: "updated",
		domain.PlotDeleted: "deleted",
	}

	messageAgeBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300}
)

type BacklogSource interface {
	Backlog(requestContext context.Context) (domain.OutboxBacklog, error)
}

type PlotCountSource interface {
	Counts(requestContext context.Context) (domain.PlotCounts, error)
}

type Stats struct {
	Backlog BacklogSource
	Plots   PlotCountSource
	Timeout time.Duration
}

type Service struct {
	serviceName     string
	registry        *prometheus.Registry
	requestDuration *prometheus.HistogramVec
	plotChanges     *prometheus.CounterVec
	previews        *prometheus.CounterVec
	previewProblems *prometheus.CounterVec
	sagas           *prometheus.CounterVec
	outboxPublished prometheus.Counter
	outboxFailures  *prometheus.CounterVec
	deliveries      *prometheus.CounterVec
	messageAge      *prometheus.HistogramVec
	now             func() time.Time
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
		serviceName:     serviceName,
		registry:        prometheus.NewRegistry(),
		requestDuration: newRequestDuration(serviceLabel),
		plotChanges:     counter("plots_changed_total", "Plots created, updated or deleted.", labelChange),
		previews:        counter("plot_previews_total", "Area previews by whether the polygon was valid.", labelResult),
		previewProblems: counter("plot_preview_problems_total", "Problems found by previews, by code.", labelReason),
		sagas: counter(
			"sagas_finished_total",
			"Sagas confirmed, cancelled (create compensated) or reverted (update compensated).",
			labelKind, labelResult, labelReason,
		),
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
		service.plotChanges,
		service.previews,
		service.previewProblems,
		service.sagas,
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

func (service *Service) PlotChanged(eventType domain.PlotEventType) {
	if change, known := changeLabels[eventType]; known {
		service.plotChanges.WithLabelValues(change).Inc()
	}
}

func (service *Service) PreviewInspected(problemCodes []string) {
	if len(problemCodes) == 0 {
		service.previews.WithLabelValues(previewValid).Inc()
		return
	}

	service.previews.WithLabelValues(previewInvalid).Inc()

	for _, code := range problemCodes {
		service.previewProblems.WithLabelValues(code).Inc()
	}
}

func (service *Service) SagaConfirmed(kind domain.SagaKind) {
	service.sagas.WithLabelValues(string(kind), sagaConfirmed, noReason).Inc()
}

func (service *Service) SagaCompensated(
	kind domain.SagaKind,
	result domain.PlotEventType,
	reason domain.CompensationReason,
) {
	outcome := sagaReverted
	if result == domain.PlotCancelled {
		outcome = sagaCancelled
	}

	service.sagas.WithLabelValues(string(kind), outcome, string(reason)).Inc()
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

func (service *Service) ObserveHealth(healthy func() bool) {
	service.registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name:        "service_health_status",
		Help:        "1 while /health reports the service healthy, 0 while it does not.",
		ConstLabels: prometheus.Labels{labelService: service.serviceName},
	}, func() float64 {
		if healthy() {
			return 1
		}

		return 0
	}))
}
