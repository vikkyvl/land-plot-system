package openmeteo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	dateLayout          = time.DateOnly
	startDateParameter  = "start_date"
	endDateParameter    = "end_date"
	dailyParameter      = "daily"
	timezoneParameter   = "timezone"
	localTimezone       = "auto"
	meanTemperature     = "temperature_2m_mean"
	minimumTemperature  = "temperature_2m_min"
	maximumTemperature  = "temperature_2m_max"
	precipitationAmount = "precipitation_sum"
)

var dailyVariables = []string{meanTemperature, minimumTemperature, maximumTemperature, precipitationAmount}

type ArchiveClient struct {
	client   *http.Client
	endpoint string
}

type archiveResponse struct {
	Daily dailySeries `json:"daily"`
}

type dailySeries struct {
	Time          []string   `json:"time"`
	Mean          []*float64 `json:"temperature_2m_mean"`
	Minimum       []*float64 `json:"temperature_2m_min"`
	Maximum       []*float64 `json:"temperature_2m_max"`
	Precipitation []*float64 `json:"precipitation_sum"`
}

func NewArchiveClient(endpoint string, timeout time.Duration, transport http.RoundTripper) *ArchiveClient {
	return &ArchiveClient{client: &http.Client{Timeout: timeout, Transport: transport}, endpoint: endpoint}
}

func (archiveClient *ArchiveClient) DailyWeather(
	requestContext context.Context,
	point domain.Coordinate,
	from, to time.Time,
) ([]domain.DailyWeather, error) {
	query := url.Values{}
	query.Set(latitudeParameter, strconv.FormatFloat(point.Latitude, 'f', coordinateDecimals, floatBits))
	query.Set(longitudeParameter, strconv.FormatFloat(point.Longitude, 'f', coordinateDecimals, floatBits))
	query.Set(startDateParameter, from.Format(dateLayout))
	query.Set(endDateParameter, to.Format(dateLayout))
	query.Set(dailyParameter, strings.Join(dailyVariables, listSeparator))
	query.Set(timezoneParameter, localTimezone)

	var body archiveResponse
	if err := getJSON(requestContext, archiveClient.client, archiveClient.endpoint, query, &body); err != nil {
		return nil, err
	}

	return body.Daily.days()
}

func (series dailySeries) days() ([]domain.DailyWeather, error) {
	count := len(series.Time)
	if len(series.Mean) != count || len(series.Minimum) != count || len(series.Maximum) != count ||
		len(series.Precipitation) != count {
		return nil, fmt.Errorf(formatSeriesLength, ErrUnexpectedResponse, count)
	}

	days := make([]domain.DailyWeather, 0, count)

	for i, day := range series.Time {
		date, err := time.Parse(dateLayout, day)
		if err != nil {
			return nil, fmt.Errorf(wrapDecodeResponse, ErrUnexpectedResponse, err)
		}

		if series.Mean[i] == nil || series.Minimum[i] == nil || series.Maximum[i] == nil ||
			series.Precipitation[i] == nil {
			continue
		}

		days = append(days, domain.DailyWeather{
			Date:            date,
			MeanC:           *series.Mean[i],
			MinC:            *series.Minimum[i],
			MaxC:            *series.Maximum[i],
			PrecipitationMm: *series.Precipitation[i],
		})
	}

	return days, nil
}
