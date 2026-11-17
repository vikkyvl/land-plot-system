package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/discovery"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

type ServiceResolver interface {
	Resolve(requestContext context.Context, serviceName string) ([]discovery.Instance, error)
}

type PlotClient struct {
	resolver    ServiceResolver
	client      *http.Client
	serviceName string
	mutex       sync.Mutex
	next        uint64
}

func NewPlotClient(resolver ServiceResolver, client *http.Client, serviceName string) *PlotClient {
	return &PlotClient{resolver: resolver, client: client, serviceName: serviceName}
}

func (plotClient *PlotClient) Get(
	requestContext context.Context,
	caller service.Caller,
	plotID string,
) (domain.PlotSnapshot, error) {
	header := authorizedHeader(caller)

	response, err := plotClient.send(requestContext, "/internal/plots/"+url.PathEscape(plotID), header)
	if err != nil {
		return domain.PlotSnapshot{}, err
	}
	defer closeBody(response)

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return domain.PlotSnapshot{}, domain.ErrPlotNotFound
	case http.StatusGone:
		return domain.PlotSnapshot{}, domain.ErrPlotDeleted
	case http.StatusUnauthorized:
		return domain.PlotSnapshot{}, domain.ErrPlotServiceRejectedToken
	case http.StatusForbidden:
		return domain.PlotSnapshot{}, forbiddenError(response)
	default:
		return domain.PlotSnapshot{}, fmt.Errorf(wrapReturned, service.ErrPlotServiceUnavailable, response.Status)
	}

	var plot plotResponse
	if err := decodeLimited(response.Body, &plot); err != nil {
		return domain.PlotSnapshot{}, err
	}

	if !plot.isValidFor(plotID) {
		return domain.PlotSnapshot{}, service.ErrInvalidPlotResponse
	}

	return plot.toDomain(), nil
}

func (plotClient *PlotClient) Trace(
	requestContext context.Context,
	caller service.Caller,
	message string,
) ([]domain.TraceHop, error) {
	header := authorizedHeader(caller)

	query := url.Values{"message": []string{message}}.Encode()

	response, err := plotClient.send(requestContext, "/internal/demo/chain?"+query, header)
	if err != nil {
		return nil, err
	}
	defer closeBody(response)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(wrapTraceReturned, service.ErrPlotServiceUnavailable, response.Status)
	}

	var trace traceResponse
	if err := decodeLimited(response.Body, &trace); err != nil {
		return nil, err
	}

	if len(trace.Path) == 0 {
		return nil, service.ErrInvalidPlotResponse
	}

	return trace.toDomain(), nil
}

func authorizedHeader(caller service.Caller) http.Header {
	header := http.Header{}
	header.Set(authorizationHeader, bearerPrefix+caller.AccessToken)
	header.Set(requestIDHeader, caller.RequestID)

	return header
}

func forbiddenError(response *http.Response) error {
	var body errorResponse
	if decodeLimited(response.Body, &body) == nil && body.Code == codeInsufficientScope {
		return domain.ErrInsufficientScope
	}

	return domain.ErrForbidden
}
