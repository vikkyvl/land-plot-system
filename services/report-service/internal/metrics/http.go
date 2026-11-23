package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelService     = "service"
	labelRoute       = "route"
	labelMethod      = "method"
	labelStatusClass = "status_class"

	unmatchedRoute     = "unmatched"
	patternSeparator   = " "
	statusClassDivisor = 100
	statusClassSuffix  = "xx"
)

var durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	if recorder.status == 0 {
		recorder.status = status
	}

	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(body []byte) (int, error) {
	if recorder.status == 0 {
		recorder.status = http.StatusOK
	}

	return recorder.ResponseWriter.Write(body)
}

func (recorder *statusRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}

func newRequestDuration(serviceLabel prometheus.Labels) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "http_server_request_duration_seconds",
		Help:        "Duration of HTTP requests served by the service, by route template.",
		Buckets:     durationBuckets,
		ConstLabels: serviceLabel,
	}, []string{labelRoute, labelMethod, labelStatusClass})
}

func observeRequests(requestDuration *prometheus.HistogramVec, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: response}

		next.ServeHTTP(recorder, request)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		requestDuration.
			WithLabelValues(routeOf(request.Pattern), request.Method, statusClass(status)).
			Observe(time.Since(started).Seconds())
	})
}

func routeOf(pattern string) string {
	if pattern == "" {
		return unmatchedRoute
	}

	if _, path, found := strings.Cut(pattern, patternSeparator); found {
		return path
	}

	return pattern
}

func statusClass(status int) string {
	return strconv.Itoa(status/statusClassDivisor) + statusClassSuffix
}
