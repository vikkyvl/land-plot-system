package metrics

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	labelService     = "service"
	labelRoute       = "route"
	labelMethod      = "method"
	labelStatusClass = "status_class"
	labelUpstream    = "upstream"
	labelOutcome     = "outcome"
	labelStatus      = "status"
	labelOrigin      = "origin"

	originGateway  = "gateway"
	originUpstream = "upstream"

	outcomeError   = "error"
	outcomeTimeout = "timeout"

	statusClassDivisor = 100
	statusClassSuffix  = "xx"
)

var (
	durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

	rejectedStatuses = map[int]bool{
		http.StatusUnauthorized:    true,
		http.StatusForbidden:       true,
		http.StatusTooManyRequests: true,
	}
)

type Gateway struct {
	registry         *prometheus.Registry
	requestDuration  *prometheus.HistogramVec
	upstreamDuration *prometheus.HistogramVec
	rejectedRequests *prometheus.CounterVec
}

func NewGateway(serviceName string) *Gateway {
	registry := prometheus.NewRegistry()
	serviceLabel := prometheus.Labels{labelService: serviceName}

	gateway := &Gateway{
		registry: registry,
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "http_server_request_duration_seconds",
			Help:        "Duration of HTTP requests served by the gateway, by route template.",
			Buckets:     durationBuckets,
			ConstLabels: serviceLabel,
		}, []string{labelRoute, labelMethod, labelStatusClass}),
		upstreamDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:        "gateway_upstream_request_duration_seconds",
			Help:        "Duration of proxied requests to each upstream service.",
			Buckets:     durationBuckets,
			ConstLabels: serviceLabel,
		}, []string{labelUpstream, labelOutcome}),
		rejectedRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:        "gateway_rejected_requests_total",
			Help:        "Responses with status 401, 403 or 429, by who produced them.",
			ConstLabels: serviceLabel,
		}, []string{labelStatus, labelOrigin}),
	}

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		gateway.requestDuration,
		gateway.upstreamDuration,
		gateway.rejectedRequests,
	)

	return gateway
}

func (gateway *Gateway) ObserveRequest(route, method string, status int, fromUpstream bool, duration time.Duration) {
	gateway.requestDuration.WithLabelValues(route, method, StatusClass(status)).Observe(duration.Seconds())

	if !rejectedStatuses[status] {
		return
	}

	origin := originGateway
	if fromUpstream {
		origin = originUpstream
	}

	gateway.rejectedRequests.WithLabelValues(strconv.Itoa(status), origin).Inc()
}

func (gateway *Gateway) ObserveUpstream(upstream string, status int, err error, duration time.Duration) {
	gateway.upstreamDuration.WithLabelValues(upstream, upstreamOutcome(status, err)).Observe(duration.Seconds())
}

func upstreamOutcome(status int, err error) string {
	var networkError net.Error

	switch {
	case err == nil:
		return StatusClass(status)
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &networkError) && networkError.Timeout():
		return outcomeTimeout
	default:
		return outcomeError
	}
}

func (gateway *Gateway) Handler() http.Handler {
	return promhttp.HandlerFor(gateway.registry, promhttp.HandlerOpts{Registry: gateway.registry})
}

func StatusClass(status int) string {
	return strconv.Itoa(status/statusClassDivisor) + statusClassSuffix
}
