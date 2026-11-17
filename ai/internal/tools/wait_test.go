package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/ai/internal/landapi"
)

func waitArguments(expectedVersion, timeoutSeconds int) json.RawMessage {
	encoded, _ := json.Marshal(map[string]any{
		"plot_id":          testPlotID,
		"expected_version": expectedVersion,
		"timeout_seconds":  timeoutSeconds,
	})

	return encoded
}

func plotWith(status string, version int64) plotResponse {
	return plotResponse{plot: landapi.Plot{ID: testPlotID, Name: "Дача", Status: status, Version: version}}
}

func TestWaitReturnsConfirmedAfterPendingPolls(t *testing.T) {
	api := &fakeAPI{
		plotResponses: []plotResponse{plotWith("PENDING", 1), plotWith("PENDING", 1), plotWith("CONFIRMED", 1)},
	}

	result := newTestToolset(api).Call(context.Background(), WaitPlotConfirmed, waitArguments(1, 10))

	outcome := decodeResult[waitResult](t, result)
	if outcome.Outcome != OutcomeConfirmed || outcome.Plot == nil || outcome.Plot.Version != 1 || api.calls != 3 {
		t.Fatalf("result = %s, calls = %d", result.Text, api.calls)
	}
}

func TestWaitReportsRolledBackUpdate(t *testing.T) {
	api := &fakeAPI{plotResponses: []plotResponse{plotWith("PENDING", 2), plotWith("CONFIRMED", 3)}}

	result := newTestToolset(api).Call(context.Background(), WaitPlotConfirmed, waitArguments(2, 10))

	outcome := decodeResult[waitResult](t, result)
	if outcome.Outcome != OutcomeRolledBack || outcome.Plot.Version != 3 {
		t.Fatalf("result = %s", result.Text)
	}
}

func TestWaitReportsCancelledPlotWhenItDisappears(t *testing.T) {
	api := &fakeAPI{plotResponses: []plotResponse{
		plotWith("PENDING", 1),
		{err: &landapi.APIError{Status: http.StatusNotFound, Code: landapi.CodePlotNotFound}},
	}}

	result := newTestToolset(api).Call(context.Background(), WaitPlotConfirmed, waitArguments(1, 10))

	if outcome := decodeResult[waitResult](t, result); outcome.Outcome != OutcomeCancelled || outcome.Plot != nil {
		t.Fatalf("result = %s", result.Text)
	}
}

func TestWaitKeepsPollingAfterTemporaryFailures(t *testing.T) {
	api := &fakeAPI{plotResponses: []plotResponse{
		{err: errNetwork},
		{err: &landapi.APIError{Status: http.StatusServiceUnavailable, Code: "service_unavailable"}},
		plotWith("CONFIRMED", 1),
	}}

	result := newTestToolset(api).Call(context.Background(), WaitPlotConfirmed, waitArguments(1, 10))

	if outcome := decodeResult[waitResult](t, result); outcome.Outcome != OutcomeConfirmed {
		t.Fatalf("result = %s", result.Text)
	}
}

func TestWaitStopsAtClientError(t *testing.T) {
	api := &fakeAPI{plotResponses: []plotResponse{
		{
			err: &landapi.APIError{
				Status:  http.StatusForbidden,
				Code:    "forbidden",
				Message: "plot belongs to another user",
			},
		},
	}}

	result := newTestToolset(api).Call(context.Background(), WaitPlotConfirmed, waitArguments(1, 10))

	if details := errorOf(t, result); details.Code != "forbidden" || api.calls != 1 {
		t.Fatalf("error = %#v, calls = %d", details, api.calls)
	}
}

func TestWaitRejectsVersionNewerThanThePlot(t *testing.T) {
	api := &fakeAPI{plotResponses: []plotResponse{plotWith("CONFIRMED", 1)}}

	result := newTestToolset(api).Call(context.Background(), WaitPlotConfirmed, waitArguments(5, 10))

	if details := errorOf(t, result); details.Code != codeVersionMismatch {
		t.Fatalf("error = %#v", details)
	}
}

func TestWaitReturnsStillPendingAtTimeout(t *testing.T) {
	api := &fakeAPI{plotResponses: []plotResponse{plotWith("PENDING", 1)}}
	started := time.Now()

	result := newTestToolset(api).Call(context.Background(), WaitPlotConfirmed, waitArguments(1, 1))

	if outcome := decodeResult[waitResult](t, result); outcome.Outcome != OutcomeStillPending {
		t.Fatalf("result = %s", result.Text)
	}

	if elapsed := time.Since(started); elapsed < time.Second || elapsed > 3*time.Second {
		t.Fatalf("waited %v", elapsed)
	}
}

func TestWaitStopsWhenTheCallerGivesUp(t *testing.T) {
	api := &fakeAPI{plotResponses: []plotResponse{plotWith("PENDING", 1)}}

	callerContext, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	result := newTestToolset(api).Call(callerContext, WaitPlotConfirmed, waitArguments(1, 40))

	if details := errorOf(t, result); details.Code != codeGatewayUnavailable {
		t.Fatalf("error = %#v", details)
	}
}
