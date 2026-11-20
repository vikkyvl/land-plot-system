package metrics

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelTarget = "target"

	TargetLandPlot           = "land-plot-service"
	TargetOpenMeteoArchive   = "open-meteo-archive"
	TargetOpenMeteoElevation = "open-meteo-elevation"

	outcomeError   = "error"
	outcomeTimeout = "timeout"
)

type observedTransport struct {
	base     http.RoundTripper
	target   string
	duration *prometheus.HistogramVec
}

func newClientDuration(serviceLabel prometheus.Labels) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "http_client_request_duration_seconds",
		Help:        "Duration of outgoing HTTP calls by target and outcome (status class, error or timeout).",
		Buckets:     durationBuckets,
		ConstLabels: serviceLabel,
	}, []string{labelTarget, labelOutcome})
}

func (transport observedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	started := time.Now()
	response, err := transport.base.RoundTrip(request)

	status := 0
	if response != nil {
		status = response.StatusCode
	}

	if err != nil {
		err = errors.Join(err, context.Cause(request.Context()))
	}

	transport.duration.WithLabelValues(transport.target, clientOutcome(status, err)).
		Observe(time.Since(started).Seconds())

	return response, err
}

func clientOutcome(status int, err error) string {
	var networkError net.Error

	switch {
	case err == nil:
		return statusClass(status)
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &networkError) && networkError.Timeout():
		return outcomeTimeout
	default:
		return outcomeError
	}
}
