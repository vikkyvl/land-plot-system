package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/api-gateway/internal/proxy"
)

func TestGatewayLogsOneLinePerRequestWithoutSecrets(t *testing.T) {
	var output bytes.Buffer

	token := "eyJ" + strings.Repeat("a", 3) + ".payload.signature"
	password := "pass" + "word-123"
	upstream := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set(proxy.HeaderServedBy, "plot-1")
		response.WriteHeader(http.StatusBadGateway)
	})
	router := NewRouter(
		logging.NewLogger(&output, "api-gateway", slog.LevelInfo),
		"api-gateway",
		upstream,
		verifierStub{userID: "user-1", scopes: []string{proxy.ScopePlotsWrite}},
		&limiterStub{allowed: true},
		testOrigins,
		&requestObserverStub{},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/plots",
		strings.NewReader(`{"name":"Plot","password":"`+password+`"}`),
	)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Cookie", "session="+token)

	response := serve(router, request)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("lines = %v, want one access line", lines)
	}

	var line map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &line); err != nil {
		t.Fatal(err)
	}

	if line["level"] != "ERROR" || line[logging.KeyRoute] != "/api/v1/plots" ||
		line[logging.KeyStatus] != float64(http.StatusBadGateway) ||
		line[logging.KeyRequestID] != response.Header().Get(proxy.HeaderRequestID) {
		t.Fatalf("line = %v", line)
	}

	for _, secret := range []string{token, password, "Authorization", "Cookie"} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("log contains %q: %s", secret, output.String())
		}
	}
}
