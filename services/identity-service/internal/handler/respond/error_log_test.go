package respond

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/identity-service/internal/logging"
)

func TestInternalErrorLogKeepsRequestIDAndNoSecrets(t *testing.T) {
	var output bytes.Buffer

	token := "eyJ" + strings.Repeat("x", 3) + ".payload.signature"
	password := "pass" + "word-123"
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/anything",
		strings.NewReader(`{"password":"`+password+`"}`),
	)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Cookie", "session="+token)
	request = request.WithContext(logging.WithRequestID(request.Context(), "req-9"))

	NewWriter(logging.NewLogger(&output, "identity-service", slog.LevelInfo)).
		Error(httptest.NewRecorder(), request, errors.New("database is down"))

	logged := output.String()
	if !strings.Contains(logged, `"request_id":"req-9"`) || !strings.Contains(logged, `"level":"ERROR"`) {
		t.Fatalf("log = %s", logged)
	}

	for _, secret := range []string{token, password, "Authorization", "Cookie"} {
		if strings.Contains(logged, secret) {
			t.Fatalf("log contains %q: %s", secret, logged)
		}
	}
}
