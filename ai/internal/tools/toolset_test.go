package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/vleshchenko/land-plot-system/ai/internal/landapi"
)

func TestToolsetKeepsOnlyKnownAllowedTools(t *testing.T) {
	toolset := NewToolset(&fakeAPI{}, Options{}, GetPlot, "delete_plot", CreateReport, GetPlot)

	if names := toolset.Names(); !slices.Equal(names, []string{GetPlot, CreateReport}) {
		t.Fatalf("Names() = %v", names)
	}

	definitions := toolset.Definitions()
	if len(definitions) != 2 || definitions[0].Name != GetPlot || !definitions[0].ReadOnly || definitions[1].ReadOnly {
		t.Fatalf("Definitions() = %#v", definitions)
	}
}

func TestEverySchemaRejectsUnknownFields(t *testing.T) {
	for _, definition := range NewToolset(&fakeAPI{}, Options{}, AllNames...).Definitions() {
		if definition.InputSchema["type"] != "object" || definition.InputSchema["additionalProperties"] != false ||
			definition.Description == "" {
			t.Fatalf("%s schema = %#v", definition.Name, definition.InputSchema)
		}

		if _, err := json.Marshal(definition.InputSchema); err != nil {
			t.Fatalf("%s schema is not JSON: %v", definition.Name, err)
		}
	}
}

func TestPolygonToolsTellTheModelNotToRepeatTheFirstPoint(t *testing.T) {
	for _, name := range []string{CreatePlot, UpdatePlot} {
		if !strings.Contains(catalog[name].Description, "do not repeat the first point") {
			t.Fatalf("%s description = %q", name, catalog[name].Description)
		}
	}
}

func TestCallRejectsToolOutsideTheAllowlist(t *testing.T) {
	api := &fakeAPI{}
	toolset := NewToolset(api, Options{}, ConsumerNames...)

	result := toolset.Call(
		context.Background(),
		CreatePlot,
		json.RawMessage(`{"name":"Дача","points":`+validPoints+`}`),
	)

	if details := errorOf(t, result); details.Code != codeUnknownTool || api.calls != 0 {
		t.Fatalf("error = %#v, API calls = %d", details, api.calls)
	}
}

func TestListPlotsFiltersByNameIgnoringCase(t *testing.T) {
	api := &fakeAPI{plots: []landapi.Plot{
		{ID: testPlotID, Name: "Дача в Ірпені", Status: "CONFIRMED", Version: 2, AreaM2: 7300},
		{ID: "other", Name: "Город на Оболоні", Status: "CONFIRMED", Version: 1},
	}}

	result := newTestToolset(api).Call(context.Background(), ListPlots, json.RawMessage(`{"name_contains":"ДАЧА"}`))

	list := decodeResult[plotListResult](t, result)
	if list.Count != 1 || list.Plots[0].ID != testPlotID || list.Plots[0].Version != 2 {
		t.Fatalf("list = %#v", list)
	}
}

func TestPlotResultContainsOnlySummaryFields(t *testing.T) {
	api := &fakeAPI{plotResponses: []plotResponse{{plot: landapi.Plot{
		ID: testPlotID, Name: "Дача", Status: "CONFIRMED", Version: 1, AreaM2: 7300, PerimeterM: 350,
		Points: []landapi.Coordinate{{Latitude: 50, Longitude: 30}},
	}}}}

	result := newTestToolset(api).Call(context.Background(), GetPlot, json.RawMessage(`{"plot_id":"`+testPlotID+`"}`))

	fields := decodeResult[map[string]any](t, result)
	for _, absent := range []string{"points", "owner_id", "area_hectares", "created_at"} {
		if _, present := fields[absent]; present {
			t.Fatalf("result contains %s: %s", absent, result.Text)
		}
	}

	if fields["area_m2"] != 7300.0 || fields["status"] != "CONFIRMED" {
		t.Fatalf("result = %s", result.Text)
	}
}

func TestReportResultContainsHectares(t *testing.T) {
	api := &fakeAPI{report: landapi.Report{
		ID: testReportID, PlotID: testPlotID, PlotName: "Дача", PlotVersion: 1, AreaM2: 7300, AreaHectares: 0.73,
	}}

	result := newTestToolset(
		api,
	).Call(context.Background(), CreateReport, json.RawMessage(`{"plot_id":"`+testPlotID+`"}`))

	report := decodeResult[reportSummary](t, result)
	if report.ID != testReportID || report.AreaHectares != 0.73 || report.PlotName != "Дача" {
		t.Fatalf("report = %#v", report)
	}
}

func TestAPIErrorIsReturnedToTheModel(t *testing.T) {
	api := &fakeAPI{err: &landapi.APIError{
		Status: http.StatusUnprocessableEntity, Code: "self_intersection", Message: "polygon must not intersect itself",
	}}

	result := newTestToolset(api).Call(
		context.Background(),
		CreatePlot,
		json.RawMessage(`{"name":"Дача","points":`+validPoints+`}`),
	)

	details := errorOf(t, result)
	if details.Code != "self_intersection" || details.Status != http.StatusUnprocessableEntity {
		t.Fatalf("error = %#v", details)
	}
}

func TestTransportErrorDoesNotLeakDetails(t *testing.T) {
	result := newTestToolset(&fakeAPI{err: errNetwork}).Call(context.Background(), ListPlots, nil)

	details := errorOf(t, result)
	if details.Code != codeGatewayUnavailable || strings.Contains(result.Text, "10.0.0.5") {
		t.Fatalf("result = %s", result.Text)
	}
}

func TestLongResultIsTruncatedToValidJSON(t *testing.T) {
	plots := make([]landapi.Plot, 40)
	for i := range plots {
		plots[i] = landapi.Plot{ID: fmt.Sprintf("plot-%d", i), Name: "Ділянка з довгою назвою номер " + strconv.Itoa(i)}
	}

	toolset := NewToolset(&fakeAPI{plots: plots}, Options{MaxResultBytes: 300}, ListPlots)

	result := toolset.Call(context.Background(), ListPlots, nil)

	truncated := decodeResult[truncatedResult](t, result)
	if len(result.Text) > 300 || !truncated.Truncated || truncated.Partial == "" ||
		!utf8.ValidString(truncated.Partial) {
		t.Fatalf("result (%d bytes) = %s", len(result.Text), result.Text)
	}
}
