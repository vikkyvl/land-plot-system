package proxy

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"time"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/discovery"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/logging"
)

type ServiceResolver interface {
	Resolve(requestContext context.Context, serviceName string) ([]discovery.Instance, error)
}

type Balancer interface {
	Select(service string, instances []discovery.Instance) (discovery.Instance, bool)
}

type upstreamContextKey struct{}

type upstream struct {
	service  string
	instance discovery.Instance
}

type Gateway struct {
	resolver        ServiceResolver
	balancer        Balancer
	logger          *slog.Logger
	upstreamTimeout time.Duration
	instanceID      string
	proxy           *httputil.ReverseProxy
}

func NewGateway(
	resolver ServiceResolver,
	balancer Balancer,
	logger *slog.Logger,
	upstreamTimeout time.Duration,
	instanceID string,
) *Gateway {
	gateway := &Gateway{
		resolver:        resolver,
		balancer:        balancer,
		logger:          logger,
		upstreamTimeout: upstreamTimeout,
		instanceID:      instanceID,
	}
	gateway.proxy = &httputil.ReverseProxy{Director: direct, ErrorHandler: gateway.handleProxyError}

	return gateway
}

func (gateway *Gateway) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	matched, ok := findRoute(request.URL.Path)
	if !ok {
		WriteError(response, request, ErrRouteNotFound)
		return
	}

	instance, ok := gateway.selectInstance(request, matched.service)
	if !ok {
		WriteError(response, request, ErrServiceUnavailable)
		return
	}

	request.Header.Set(HeaderGatewayInstance, gateway.instanceID)
	response.Header().Set(HeaderServedBy, instance.ID)

	upstreamContext, cancel := context.WithTimeout(request.Context(), gateway.upstreamTimeout)
	defer cancel()

	upstreamContext = context.WithValue(
		upstreamContext,
		upstreamContextKey{},
		upstream{service: matched.service, instance: instance},
	)
	gateway.proxy.ServeHTTP(response, request.WithContext(upstreamContext))
}

func (gateway *Gateway) selectInstance(request *http.Request, service string) (discovery.Instance, bool) {
	instances, err := gateway.resolver.Resolve(request.Context(), service)
	if err != nil {
		gateway.logger.Error(
			logResolveService,
			logging.KeyServiceName, service,
			logging.KeyError, err,
			logging.KeyRequestID, request.Header.Get(HeaderRequestID),
		)

		return discovery.Instance{}, false
	}

	return gateway.balancer.Select(service, instances)
}

func direct(request *http.Request) {
	target := request.Context().Value(upstreamContextKey{}).(upstream).instance
	request.URL.Scheme = "http"
	request.URL.Host = net.JoinHostPort(target.Address, strconv.Itoa(target.Port))

	if _, ok := request.Header["User-Agent"]; !ok {
		request.Header.Set("User-Agent", "")
	}
}

func (gateway *Gateway) handleProxyError(response http.ResponseWriter, request *http.Request, err error) {
	target, _ := request.Context().Value(upstreamContextKey{}).(upstream)
	gateway.logger.Error(
		logProxyUpstreamRequest,
		logging.KeyServiceName, target.service,
		logging.KeyInstanceID, target.instance.ID,
		logging.KeyError, err,
		logging.KeyRequestID, request.Header.Get(HeaderRequestID),
	)
	WriteError(response, request, ErrBadGateway)
}
