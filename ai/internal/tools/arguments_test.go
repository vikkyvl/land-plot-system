package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const validPoints = `[{"lat":50.5203,"lon":30.2471},{"lat":50.5203,"lon":30.2486},{"lat":50.5212,"lon":30.2486}]`

func manyPoints(count int) string {
	points := make([]string, count)
	for i := range points {
		points[i] = fmt.Sprintf(`{"lat":50.%04d,"lon":30.%04d}`, i, i*7%10000)
	}

	return "[" + strings.Join(points, ",") + "]"
}

func TestInvalidArgumentsAreRejectedBeforeTheAPICall(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		arguments string
	}{
		{"unknown field", ListPlots, `{"name_contains":"Дача","owner":"x"}`},
		{"not an object", ListPlots, `["Дача"]`},
		{"trailing JSON", GetPlot, `{"plot_id":"` + testPlotID + `"} {}`},
		{"plot id is not a UUID", GetPlot, `{"plot_id":"../plots"}`},
		{"missing plot id", GetPlot, `{}`},
		{"report id is not a UUID", GetReport, `{"report_id":"42"}`},
		{"report plot id is not a UUID", CreateReport, `{"plot_id":"plot-1"}`},
		{"missing name", CreatePlot, `{"points":` + validPoints + `}`},
		{"blank name", CreatePlot, `{"name":"   ","points":` + validPoints + `}`},
		{"name too long", CreatePlot, `{"name":"` + strings.Repeat("ї", 121) + `","points":` + validPoints + `}`},
		{"name with a control character", CreatePlot, `{"name":"Дача\u0007","points":` + validPoints + `}`},
		{"two points", CreatePlot, `{"name":"Дача","points":[{"lat":50,"lon":30},{"lat":51,"lon":31}]}`},
		{"too many points", CreatePlot, `{"name":"Дача","points":` + manyPoints(101) + `}`},
		{
			"latitude 200",
			CreatePlot,
			`{"name":"Дача","points":[{"lat":200,"lon":30},{"lat":51,"lon":31},{"lat":52,"lon":30}]}`,
		},
		{
			"longitude -181",
			CreatePlot,
			`{"name":"Дача","points":[{"lat":50,"lon":-181},{"lat":51,"lon":31},{"lat":52,"lon":30}]}`,
		},
		{
			"missing longitude",
			CreatePlot,
			`{"name":"Дача","points":[{"lat":50},{"lat":51,"lon":31},{"lat":52,"lon":30}]}`,
		},
		{
			"unknown point field",
			CreatePlot,
			`{"name":"Дача","points":[{"lat":50,"lon":30,"alt":1},{"lat":51,"lon":31},{"lat":52,"lon":30}]}`,
		},
		{
			"closed polygon",
			CreatePlot,
			`{"name":"Дача","points":[{"lat":50,"lon":30},{"lat":51,"lon":31},{"lat":52,"lon":30},{"lat":50,"lon":30}]}`,
		},
		{"update without changes", UpdatePlot, `{"plot_id":"` + testPlotID + `"}`},
		{"update with a bad id", UpdatePlot, `{"plot_id":"x","name":"Дача"}`},
		{
			"update with two points",
			UpdatePlot,
			`{"plot_id":"` + testPlotID + `","points":[{"lat":50,"lon":30},{"lat":51,"lon":31}]}`,
		},
		{"wait without version", WaitPlotConfirmed, `{"plot_id":"` + testPlotID + `","timeout_seconds":10}`},
		{
			"wait with version 0",
			WaitPlotConfirmed,
			`{"plot_id":"` + testPlotID + `","expected_version":0,"timeout_seconds":10}`,
		},
		{
			"wait for 0 seconds",
			WaitPlotConfirmed,
			`{"plot_id":"` + testPlotID + `","expected_version":1,"timeout_seconds":0}`,
		},
		{
			"wait for 41 seconds",
			WaitPlotConfirmed,
			`{"plot_id":"` + testPlotID + `","expected_version":1,"timeout_seconds":41}`,
		},
		{
			"wait with fractional timeout",
			WaitPlotConfirmed,
			`{"plot_id":"` + testPlotID + `","expected_version":1,"timeout_seconds":1.5}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			api := &fakeAPI{}

			result := newTestToolset(api).Call(context.Background(), test.tool, json.RawMessage(test.arguments))

			if details := errorOf(t, result); details.Code != codeInvalidArguments || details.Message == "" {
				t.Fatalf("error = %#v", details)
			}

			if api.calls != 0 {
				t.Fatalf("API was called %d times", api.calls)
			}
		})
	}
}

func TestCreatePlotNormalizesTheName(t *testing.T) {
	api := &fakeAPI{}
	name := strings.Repeat("ї", 118)

	result := newTestToolset(api).Call(
		context.Background(),
		CreatePlot,
		json.RawMessage(`{"name":"  `+name+`  ","points":`+validPoints+`}`),
	)
	if result.IsError {
		t.Fatalf("Call() = %s", result.Text)
	}

	if api.draft.Name != name || len(api.draft.Points) != 3 || api.draft.Points[1].Longitude != 30.2486 {
		t.Fatalf("draft = %#v", api.draft)
	}
}

func TestCreatePlotAcceptsHundredPoints(t *testing.T) {
	api := &fakeAPI{}

	result := newTestToolset(api).Call(
		context.Background(),
		CreatePlot,
		json.RawMessage(`{"name":"Дача","points":`+manyPoints(100)+`}`),
	)
	if result.IsError || len(api.draft.Points) != 100 {
		t.Fatalf("Call() = %s, points = %d", result.Text, len(api.draft.Points))
	}
}

func TestUpdatePlotSendsOnlyTheName(t *testing.T) {
	api := &fakeAPI{}

	result := newTestToolset(api).Call(
		context.Background(),
		UpdatePlot,
		json.RawMessage(`{"plot_id":"`+testPlotID+`","name":" Дача в Ірпені, вул. Садова 12 "}`),
	)
	if result.IsError {
		t.Fatalf("Call() = %s", result.Text)
	}

	if api.changes.Name == nil || *api.changes.Name != "Дача в Ірпені, вул. Садова 12" || api.changes.Points != nil {
		t.Fatalf("changes = %#v", api.changes)
	}
}

func TestEmptyArgumentsMeanAnEmptyObject(t *testing.T) {
	result := newTestToolset(&fakeAPI{}).Call(context.Background(), ListPlots, nil)
	if result.IsError {
		t.Fatalf("Call() = %s", result.Text)
	}
}
