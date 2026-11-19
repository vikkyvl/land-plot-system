package logging

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	HeaderRequestID = "X-Request-ID"
	HealthRoute     = "/health"
	UnmatchedRoute  = "unmatched"

	logHTTPRequest   = "http request"
	patternSeparator = " "
)

type RouteOf func(request *http.Request) string

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

func (recorder *statusRecorder) Flush() {
	if flusher, ok := recorder.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func AccessLog(logger *slog.Logger, routeOf RouteOf, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()

		requestID := RequestID(request.Context())
		if requestID == "" {
			requestID = request.Header.Get(HeaderRequestID)
		}

		if requestID == "" {
			requestID = NewRequestID()
		}

		request.Header.Set(HeaderRequestID, requestID)
		request = request.WithContext(WithRequestID(request.Context(), requestID))
		recorder := &statusRecorder{ResponseWriter: response}

		next.ServeHTTP(recorder, request)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		route := routeOf(request)
		logger.Log(
			request.Context(),
			accessLevel(route, status),
			logHTTPRequest,
			KeyMethod, request.Method,
			KeyRoute, route,
			KeyStatus, status,
			KeyDurationMS, time.Since(started).Milliseconds(),
		)
	})
}

func PatternRoute(request *http.Request) string {
	pattern := request.Pattern
	if pattern == "" {
		return UnmatchedRoute
	}

	if _, path, found := strings.Cut(pattern, patternSeparator); found {
		return path
	}

	return pattern
}

func accessLevel(route string, status int) slog.Level {
	switch {
	case route == HealthRoute && status < http.StatusBadRequest:
		return slog.LevelDebug
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
