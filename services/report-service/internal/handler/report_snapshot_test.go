package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func getReportBody(t *testing.T, report domain.Report) map[string]json.RawMessage {
	t.Helper()

	response := serve(
		newTestRouter(&reportServiceStub{report: report}),
		authorizedRequest(http.MethodGet, "/api/v1/reports/report-1", ""),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return body
}

func TestReportShowsAccuracyTerrainAndClimate(t *testing.T) {
	worstSide := 2
	body := getReportBody(t, domain.Report{
		ID:        "report-1",
		CreatedAt: time.Now(),
		Accuracy: &domain.AreaAccuracy{
			AreaMarginM2:     520,
			ScaleShare:       0.4,
			ObstructionShare: 0.6,
			WorstSide:        &worstSide,
			Basis:            "assumed",
		},
		Terrain: &domain.Terrain{Status: domain.DataAvailable, SlopeDegrees: 10, SurfaceAreaM2: 25_000, Coarse: true},
		Climate: &domain.Climate{
			Status: domain.DataAvailable, FirstYear: 2016, LastYear: 2025, GrowingDays: 239,
			Seasons:         []domain.SeasonClimate{{Season: domain.SeasonWinter, MeanC: -1}},
			LastSpringFrost: &domain.MonthDay{Month: time.April, Day: 5},
		},
	})

	checks := map[string][]string{
		"accuracy": {
			`"area_m2_margin":520`,
			`"contributions":{"scale":0.4,"obstructions":0.6}`,
			`"worst_side":2`,
			`"basis":"assumed"`,
		},
		"terrain": {
			`"status":"available"`,
			`"surface_area_hectares":2.5`,
			`"coarse":true`,
			`"source":"Open-Meteo.com (CC BY 4.0), Copernicus DEM GLO-90"`,
		},
		"climate": {
			`"first_year":2016`,
			`"seasons":[{"season":"winter","mean_c":-1`,
			`"last_spring_frost":"04-05"`,
			`"first_autumn_frost":null`,
			`"growing_days":239`,
			`"source":"Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5"`,
		},
	}
	for block, fragments := range checks {
		for _, fragment := range fragments {
			if !strings.Contains(string(body[block]), fragment) {
				t.Errorf("%s = %s, want %s", block, body[block], fragment)
			}
		}
	}
}

func TestOlderReportHasEmptyBlocks(t *testing.T) {
	body := getReportBody(t, domain.Report{ID: "report-1", CreatedAt: time.Now()})

	for _, block := range []string{"accuracy", "terrain", "climate"} {
		if string(body[block]) != "null" {
			t.Errorf("%s = %s, want null", block, body[block])
		}
	}
}

func TestUnavailableBlocksKeepTheirStatus(t *testing.T) {
	period := domain.ClimatePeriod{FirstYear: 2016, LastYear: 2025}
	body := getReportBody(t, domain.Report{ID: "report-1", CreatedAt: time.Now()}.
		WithSurroundings(domain.UnavailableTerrain(), domain.UnavailableClimate(period)))

	if !strings.Contains(string(body["terrain"]), `"status":"unavailable"`) ||
		!strings.Contains(string(body["climate"]), `"status":"unavailable"`) ||
		!strings.Contains(string(body["climate"]), `"seasons":[]`) {
		t.Fatalf("terrain = %s climate = %s", body["terrain"], body["climate"])
	}
}
