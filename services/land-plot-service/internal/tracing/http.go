package tracing

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	serverOperation = "http.server"
	routeAttribute  = "http.route"
	healthPath      = "/health"
	nameSeparator   = " "
)

type RouteOf func(request *http.Request) string

func Server(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, serverOperation, otelhttp.WithFilter(func(request *http.Request) bool {
		return request.URL.Path != healthPath
	}))
}

func NameByRoute(routeOf RouteOf, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		next.ServeHTTP(response, request)

		route := routeOf(request)
		span := trace.SpanFromContext(request.Context())
		span.SetName(request.Method + nameSeparator + route)
		span.SetAttributes(attribute.String(routeAttribute, route))
	})
}

func Transport(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(base, otelhttp.WithSpanNameFormatter(clientSpanName))
}

func TransportWithoutPropagation(base http.RoundTripper) http.RoundTripper {
	return otelhttp.NewTransport(
		base,
		otelhttp.WithSpanNameFormatter(clientSpanName),
		otelhttp.WithPropagators(propagation.NewCompositeTextMapPropagator()),
	)
}

func clientSpanName(_ string, request *http.Request) string {
	return request.Method + nameSeparator + request.URL.Host
}
