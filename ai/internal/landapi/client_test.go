package landapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testToken    = "token-1"
	testPlotID   = "4b6f1a52-38f5-4c3e-9e8b-1d2f3a4b5c6d"
	testReportID = "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"
)

var testNow = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

type fakeGateway struct {
	logins        atomic.Int32
	registrations atomic.Int32
	mux           *http.ServeMux
}

func newFakeGateway() *fakeGateway {
	gateway := &fakeGateway{mux: http.NewServeMux()}
	gateway.mux.HandleFunc("POST /api/v1/auth/login", func(response http.ResponseWriter, _ *http.Request) {
		number := gateway.logins.Add(1)
		writeJSON(response, http.StatusOK, map[string]any{
			"access_token": "token-" + strconv.Itoa(int(number)),
			"token_type":   "Bearer",
			"expires_at":   testNow.Add(time.Hour),
		})
	})

	return gateway
}

func (gateway *fakeGateway) start(t *testing.T) *Client {
	t.Helper()

	server := httptest.NewServer(gateway.mux)
	t.Cleanup(server.Close)

	client := New(Config{
		BaseURL:     server.URL,
		Timeout:     time.Second,
		Credentials: Credentials{Email: "agent@example.com", Password: "AgentPassword123"},
	})
	client.now = func() time.Time { return testNow }

	return client
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}

func writeError(response http.ResponseWriter, status int, code string) {
	writeJSON(response, status, map[string]string{"code": code, "message": code + " message", "request_id": "r"})
}

func TestAuthenticateRegistersUnknownUserAndLogsIn(t *testing.T) {
	gateway := &fakeGateway{mux: http.NewServeMux()}
	registered := atomic.Bool{}

	gateway.mux.HandleFunc("POST /api/v1/auth/register", func(response http.ResponseWriter, request *http.Request) {
		gateway.registrations.Add(1)

		var credentials Credentials
		if err := json.NewDecoder(request.Body).
			Decode(&credentials); err != nil ||
			credentials.Email != "agent@example.com" {
			writeError(response, http.StatusBadRequest, "invalid_body")
			return
		}

		registered.Store(true)
		writeJSON(response, http.StatusCreated, map[string]string{"id": "u"})
	})
	gateway.mux.HandleFunc("POST /api/v1/auth/login", func(response http.ResponseWriter, _ *http.Request) {
		gateway.logins.Add(1)

		if !registered.Load() {
			writeError(response, http.StatusUnauthorized, CodeLoginRejected)
			return
		}

		writeJSON(
			response,
			http.StatusOK,
			map[string]any{"access_token": testToken, "expires_at": testNow.Add(time.Hour)},
		)
	})

	client := gateway.start(t)

	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if gateway.registrations.Load() != 1 || gateway.logins.Load() != 2 || client.token != testToken {
		t.Fatalf("registrations = %d, logins = %d, token = %q",
			gateway.registrations.Load(), gateway.logins.Load(), client.token)
	}
}

func TestAuthenticateIgnoresUserThatAlreadyExists(t *testing.T) {
	gateway := &fakeGateway{mux: http.NewServeMux()}
	gateway.mux.HandleFunc("POST /api/v1/auth/register", func(response http.ResponseWriter, _ *http.Request) {
		gateway.registrations.Add(1)
		writeError(response, http.StatusConflict, CodeUserExists)
	})
	gateway.mux.HandleFunc("POST /api/v1/auth/login", func(response http.ResponseWriter, _ *http.Request) {
		if gateway.logins.Add(1) == 1 {
			writeError(response, http.StatusUnauthorized, CodeLoginRejected)
			return
		}

		writeJSON(
			response,
			http.StatusOK,
			map[string]any{"access_token": testToken, "expires_at": testNow.Add(time.Hour)},
		)
	})

	if err := gateway.start(t).Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if gateway.registrations.Load() != 1 || gateway.logins.Load() != 2 {
		t.Fatalf("registrations = %d, logins = %d", gateway.registrations.Load(), gateway.logins.Load())
	}
}

func TestAuthenticateReturnsLoginFailure(t *testing.T) {
	gateway := &fakeGateway{mux: http.NewServeMux()}
	gateway.mux.HandleFunc("POST /api/v1/auth/login", func(response http.ResponseWriter, _ *http.Request) {
		writeError(response, http.StatusServiceUnavailable, "service_unavailable")
	})

	err := gateway.start(t).Authenticate(context.Background())

	apiError, ok := errors.AsType[*APIError](err)
	if !ok || apiError.Status != http.StatusServiceUnavailable || gateway.registrations.Load() != 0 {
		t.Fatalf("Authenticate() error = %v", err)
	}
}

func TestClientReusesTokenUntilItIsAboutToExpire(t *testing.T) {
	gateway := newFakeGateway()
	gateway.mux.HandleFunc("GET /api/v1/plots", func(response http.ResponseWriter, _ *http.Request) {
		writeJSON(response, http.StatusOK, []Plot{})
	})

	client := gateway.start(t)
	for range 2 {
		if _, err := client.ListPlots(context.Background()); err != nil {
			t.Fatalf("ListPlots() error = %v", err)
		}
	}

	if gateway.logins.Load() != 1 {
		t.Fatalf("logins = %d, want 1", gateway.logins.Load())
	}

	client.now = func() time.Time { return testNow.Add(time.Hour - refreshMargin) }

	if _, err := client.ListPlots(context.Background()); err != nil {
		t.Fatalf("ListPlots() error = %v", err)
	}

	if gateway.logins.Load() != 2 {
		t.Fatalf("logins after expiry = %d, want 2", gateway.logins.Load())
	}
}

func TestClientLogsInAgainAfterUnauthorizedResponse(t *testing.T) {
	gateway := newFakeGateway()
	requests := atomic.Int32{}

	gateway.mux.HandleFunc("GET /api/v1/plots", func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)

		if request.Header.Get("Authorization") == "Bearer token-1" {
			writeError(response, http.StatusUnauthorized, "unauthorized")
			return
		}

		writeJSON(response, http.StatusOK, []Plot{{ID: testPlotID, Name: "Plot"}})
	})

	plots, err := gateway.start(t).ListPlots(context.Background())
	if err != nil || len(plots) != 1 || plots[0].ID != testPlotID {
		t.Fatalf("ListPlots() = %#v, %v", plots, err)
	}

	if gateway.logins.Load() != 2 || requests.Load() != 2 {
		t.Fatalf("logins = %d, requests = %d", gateway.logins.Load(), requests.Load())
	}
}

func TestClientRetriesUnauthorizedRequestOnlyOnce(t *testing.T) {
	gateway := newFakeGateway()
	gateway.mux.HandleFunc("GET /api/v1/plots", func(response http.ResponseWriter, _ *http.Request) {
		writeError(response, http.StatusUnauthorized, "unauthorized")
	})

	_, err := gateway.start(t).ListPlots(context.Background())

	apiError, ok := errors.AsType[*APIError](err)
	if !ok || apiError.Status != http.StatusUnauthorized || gateway.logins.Load() != 2 {
		t.Fatalf("ListPlots() error = %v, logins = %d", err, gateway.logins.Load())
	}
}

func TestCreatePlotSendsTokenAndCoordinates(t *testing.T) {
	gateway := newFakeGateway()
	gateway.mux.HandleFunc("POST /api/v1/plots", func(response http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeError(response, http.StatusBadRequest, "invalid_body")
			return
		}

		points, _ := body["points"].([]any)
		first, _ := points[0].(map[string]any)

		if request.Header.Get("Authorization") != "Bearer token-1" ||
			request.Header.Get("Content-Type") != contentTypeJSON || first["lat"] != 50.5 || first["lon"] != 30.2 {
			writeError(response, http.StatusBadRequest, "unexpected_request")
			return
		}

		writeJSON(response, http.StatusCreated, Plot{
			ID: testPlotID, Name: "Дача", AreaM2: 1200, Version: 1, Status: "PENDING",
		})
	})

	plot, err := gateway.start(t).CreatePlot(context.Background(), PlotDraft{
		Name:   "Дача",
		Points: []Coordinate{{50.5, 30.2}, {50.5, 30.3}, {50.6, 30.3}},
	})
	if err != nil {
		t.Fatalf("CreatePlot() error = %v", err)
	}

	if plot.ID != testPlotID || plot.Status != "PENDING" || plot.Version != 1 || plot.AreaM2 != 1200 {
		t.Fatalf("CreatePlot() = %#v", plot)
	}
}

func TestUpdatePlotSendsOnlyProvidedFields(t *testing.T) {
	gateway := newFakeGateway()
	gateway.mux.HandleFunc("PATCH /api/v1/plots/{id}", func(response http.ResponseWriter, request *http.Request) {
		var body map[string]any

		_ = json.NewDecoder(request.Body).Decode(&body)

		if _, hasPoints := body["points"]; hasPoints || body["name"] != "Нова" ||
			request.PathValue("id") != testPlotID {
			writeError(response, http.StatusBadRequest, "unexpected_request")
			return
		}

		writeJSON(response, http.StatusOK, Plot{ID: testPlotID, Name: "Нова", Version: 2, Status: "PENDING"})
	})

	name := "Нова"

	plot, err := gateway.start(t).UpdatePlot(context.Background(), testPlotID, PlotChanges{Name: &name})
	if err != nil || plot.Version != 2 {
		t.Fatalf("UpdatePlot() = %#v, %v", plot, err)
	}
}

func TestClientReturnsGatewayErrorCode(t *testing.T) {
	gateway := newFakeGateway()
	gateway.mux.HandleFunc("POST /api/v1/plots", func(response http.ResponseWriter, _ *http.Request) {
		writeError(response, http.StatusUnprocessableEntity, "self_intersection")
	})
	gateway.mux.HandleFunc("GET /api/v1/plots/{id}", func(response http.ResponseWriter, _ *http.Request) {
		writeError(response, http.StatusNotFound, CodePlotNotFound)
	})

	client := gateway.start(t)

	_, err := client.CreatePlot(context.Background(), PlotDraft{Name: "Plot"})

	apiError, ok := errors.AsType[*APIError](err)
	if !ok || apiError.Status != http.StatusUnprocessableEntity || apiError.Code != "self_intersection" ||
		apiError.Message != "self_intersection message" {
		t.Fatalf("CreatePlot() error = %#v", err)
	}

	if _, err := client.GetPlot(context.Background(), testPlotID); !IsNotFound(err) {
		t.Fatalf("GetPlot() error = %v, want not found", err)
	}
}

func TestClientHandlesErrorWithoutJSONBody(t *testing.T) {
	gateway := newFakeGateway()
	gateway.mux.HandleFunc("GET /api/v1/plots", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusBadGateway)
		_, _ = response.Write([]byte("<html>bad gateway</html>"))
	})

	_, err := gateway.start(t).ListPlots(context.Background())

	apiError, ok := errors.AsType[*APIError](err)
	if !ok || apiError.Status != http.StatusBadGateway || apiError.Code != "" ||
		apiError.Message != unexpectedStatusText {
		t.Fatalf("ListPlots() error = %#v", err)
	}
}

func TestClientEscapesPathIdentifiers(t *testing.T) {
	gateway := newFakeGateway()
	escapedPath := make(chan string, 1)

	gateway.mux.HandleFunc("/api/v1/plots/", func(response http.ResponseWriter, request *http.Request) {
		escapedPath <- request.URL.EscapedPath()

		writeError(response, http.StatusBadRequest, "invalid_plot_id")
	})

	_, _ = gateway.start(t).GetPlot(context.Background(), "a/b?c")

	if path := <-escapedPath; path != "/api/v1/plots/a%2Fb%3Fc" {
		t.Fatalf("escaped path = %q", path)
	}
}

func TestReportsAreCreatedAndRead(t *testing.T) {
	gateway := newFakeGateway()
	report := Report{ID: testReportID, PlotID: testPlotID, PlotName: "Дача", AreaHectares: 0.12, PlotVersion: 1}

	gateway.mux.HandleFunc("POST /api/v1/reports", func(response http.ResponseWriter, request *http.Request) {
		var body createReportRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body.PlotID != testPlotID {
			writeError(response, http.StatusBadRequest, "invalid_body")
			return
		}

		writeJSON(response, http.StatusCreated, report)
	})
	gateway.mux.HandleFunc("GET /api/v1/reports/{id}", func(response http.ResponseWriter, request *http.Request) {
		if request.PathValue("id") != testReportID {
			writeError(response, http.StatusNotFound, "report_not_found")
			return
		}

		writeJSON(response, http.StatusOK, report)
	})

	client := gateway.start(t)

	created, err := client.CreateReport(context.Background(), testPlotID)
	if err != nil || created.ID != testReportID || created.AreaHectares != 0.12 {
		t.Fatalf("CreateReport() = %#v, %v", created, err)
	}

	read, err := client.GetReport(context.Background(), testReportID)
	if err != nil || read.PlotName != "Дача" {
		t.Fatalf("GetReport() = %#v, %v", read, err)
	}
}

func TestClientStopsAtTimeout(t *testing.T) {
	gateway := newFakeGateway()
	gateway.mux.HandleFunc("GET /api/v1/plots", func(response http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-time.After(time.Second):
		}

		writeJSON(response, http.StatusOK, []Plot{})
	})

	client := gateway.start(t)
	client.httpClient.Timeout = 50 * time.Millisecond

	_, err := client.ListPlots(context.Background())
	if err == nil || errors.As(err, new(*APIError)) || !strings.Contains(err.Error(), "send request GET") {
		t.Fatalf("ListPlots() error = %v, want a transport timeout", err)
	}
}
