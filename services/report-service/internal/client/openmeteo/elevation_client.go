package openmeteo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	MaximumElevationPoints = 100
	maximumResponseSize    = 4 << 20
	coordinateDecimals     = 6
	floatBits              = 64
	listSeparator          = ","
	latitudeParameter      = "latitude"
	longitudeParameter     = "longitude"
)

type ElevationClient struct {
	client   *http.Client
	endpoint string
}

type elevationResponse struct {
	Elevation []*float64 `json:"elevation"`
}

type errorResponse struct {
	Reason string `json:"reason"`
}

func NewElevationClient(endpoint string, timeout time.Duration, transport http.RoundTripper) *ElevationClient {
	return &ElevationClient{client: &http.Client{Timeout: timeout, Transport: transport}, endpoint: endpoint}
}

func (elevationClient *ElevationClient) Elevations(
	requestContext context.Context,
	points []domain.Coordinate,
) ([]float64, error) {
	if len(points) > MaximumElevationPoints {
		return nil, ErrTooManyElevationPoints
	}

	query := url.Values{}
	query.Set(
		latitudeParameter,
		joinCoordinates(points, func(point domain.Coordinate) float64 { return point.Latitude }),
	)
	query.Set(
		longitudeParameter,
		joinCoordinates(points, func(point domain.Coordinate) float64 { return point.Longitude }),
	)

	var body elevationResponse
	if err := getJSON(requestContext, elevationClient.client, elevationClient.endpoint, query, &body); err != nil {
		return nil, err
	}

	if len(body.Elevation) != len(points) {
		return nil, fmt.Errorf(formatValueCount, ErrUnexpectedResponse, len(body.Elevation), len(points))
	}

	elevations := make([]float64, len(points))

	for i, elevation := range body.Elevation {
		if elevation == nil {
			return nil, fmt.Errorf(formatMissingValue, ErrUnexpectedResponse, i)
		}

		elevations[i] = *elevation
	}

	return elevations, nil
}

func joinCoordinates(points []domain.Coordinate, value func(domain.Coordinate) float64) string {
	values := make([]string, len(points))
	for i, point := range points {
		values[i] = strconv.FormatFloat(value(point), 'f', coordinateDecimals, floatBits)
	}

	return strings.Join(values, listSeparator)
}

func getJSON(requestContext context.Context, client *http.Client, endpoint string, query url.Values, target any) error {
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint+"?"+query.Encode(), http.NoBody)
	if err != nil {
		return fmt.Errorf(wrapCreateRequest, err)
	}

	response, err := client.Do(request)
	if err != nil {
		var requestError *url.Error
		if errors.As(err, &requestError) {
			err = requestError.Err
		}

		return fmt.Errorf(wrapSendRequest, err)
	}
	defer func() { _ = response.Body.Close() }()

	body := io.LimitReader(response.Body, maximumResponseSize)

	if response.StatusCode != http.StatusOK {
		var problem errorResponse

		_ = json.NewDecoder(body).Decode(&problem)

		return fmt.Errorf(formatStatusReason, ErrUnexpectedResponse, response.StatusCode, problem.Reason)
	}

	if err := json.NewDecoder(body).Decode(target); err != nil {
		return fmt.Errorf(wrapDecodeResponse, ErrUnexpectedResponse, err)
	}

	return nil
}
