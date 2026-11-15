package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"strconv"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/discovery"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

const (
	maximumResponseSize = 1 << 20
	requestIDHeader     = "X-Request-ID"
)

func (plotClient *PlotClient) send(
	requestContext context.Context,
	pathAndQuery string,
	header http.Header,
) (*http.Response, error) {
	instance, err := plotClient.nextInstance(requestContext)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(
		requestContext,
		http.MethodGet,
		instanceURL(instance)+pathAndQuery,
		http.NoBody,
	)
	if err != nil {
		return nil, fmt.Errorf(wrapCreatePlotServiceRequest, err)
	}

	maps.Copy(request.Header, header)

	response, err := plotClient.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", service.ErrPlotServiceUnavailable, err)
	}

	return response, nil
}

func (plotClient *PlotClient) nextInstance(requestContext context.Context) (discovery.Instance, error) {
	instances, err := plotClient.resolver.Resolve(requestContext, plotClient.serviceName)
	if err != nil {
		return discovery.Instance{}, fmt.Errorf("%w: %w", service.ErrPlotServiceUnavailable, err)
	}

	if len(instances) == 0 {
		return discovery.Instance{}, service.ErrPlotServiceUnavailable
	}

	plotClient.mutex.Lock()
	defer plotClient.mutex.Unlock()

	instance := instances[plotClient.next%uint64(len(instances))]
	plotClient.next++

	return instance, nil
}

func instanceURL(instance discovery.Instance) string {
	return "http://" + net.JoinHostPort(instance.Address, strconv.Itoa(instance.Port))
}

func decodeLimited(body io.Reader, destination any) error {
	if err := json.NewDecoder(io.LimitReader(body, maximumResponseSize)).Decode(destination); err != nil {
		return fmt.Errorf("%w: %w", service.ErrInvalidPlotResponse, err)
	}

	return nil
}

func closeBody(response *http.Response) {
	_ = response.Body.Close()
}
