package openmeteo

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var (
	testArchivePoint = domain.Coordinate{Latitude: 50.5, Longitude: 30.5}
	testArchiveFrom  = time.Date(2015, time.December, 1, 0, 0, 0, 0, time.UTC)
	testArchiveTo    = time.Date(2025, time.December, 31, 0, 0, 0, 0, time.UTC)
)

const testArchiveBody = `{"daily":{"time":["2016-01-01","2016-01-02","2016-01-03"],` +
	`"temperature_2m_mean":[-3.5,null,-1.0],"temperature_2m_min":[-8.0,-6.0,-4.0],` +
	`"temperature_2m_max":[0.5,1.0,2.0],"precipitation_sum":[1.2,0.0,3.4]}}`

func TestDailyWeatherAsksForTheSeriesAndSkipsGaps(t *testing.T) {
	var rawQuery string

	server := elevationServer(t, http.StatusOK, testArchiveBody, &rawQuery)

	days, err := NewArchiveClient(server.URL, testTimeout, http.DefaultTransport).
		DailyWeather(context.Background(), testArchivePoint, testArchiveFrom, testArchiveTo)
	if err != nil {
		t.Fatalf("DailyWeather() error = %v", err)
	}

	want := []domain.DailyWeather{
		{
			Date:            time.Date(2016, time.January, 1, 0, 0, 0, 0, time.UTC),
			MeanC:           -3.5,
			MinC:            -8,
			MaxC:            0.5,
			PrecipitationMm: 1.2,
		},
		{
			Date:            time.Date(2016, time.January, 3, 0, 0, 0, 0, time.UTC),
			MeanC:           -1,
			MinC:            -4,
			MaxC:            2,
			PrecipitationMm: 3.4,
		},
	}
	if !reflect.DeepEqual(days, want) {
		t.Fatalf("days = %+v", days)
	}

	query, _ := url.ParseQuery(rawQuery)

	expected := url.Values{
		"latitude":   {"50.500000"},
		"longitude":  {"30.500000"},
		"start_date": {"2015-12-01"},
		"end_date":   {"2025-12-31"},
		"daily":      {"temperature_2m_mean,temperature_2m_min,temperature_2m_max,precipitation_sum"},
		"timezone":   {"auto"},
	}
	if !reflect.DeepEqual(query, expected) {
		t.Fatalf("query = %v", query)
	}
}

func TestDailyWeatherRejectsBadSeries(t *testing.T) {
	tests := map[string]string{
		"uneven series": `{"daily":{"time":["2016-01-01"],"temperature_2m_mean":[],"temperature_2m_min":[1],` +
			`"temperature_2m_max":[1],"precipitation_sum":[1]}}`,
		"bad date": `{"daily":{"time":["01.01.2016"],"temperature_2m_mean":[1],"temperature_2m_min":[1],` +
			`"temperature_2m_max":[1],"precipitation_sum":[1]}}`,
	}

	for title, body := range tests {
		server := elevationServer(t, http.StatusOK, body, nil)

		_, err := NewArchiveClient(server.URL, testTimeout, http.DefaultTransport).
			DailyWeather(context.Background(), testArchivePoint, testArchiveFrom, testArchiveTo)
		if !errors.Is(err, ErrUnexpectedResponse) {
			t.Errorf("%s: error = %v, want ErrUnexpectedResponse", title, err)
		}
	}
}
