package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testModel     = "gemini-3.5-flash-lite"
	testSignature = "c2lnbmF0dXJlLTE="
)

type scriptedReply struct {
	status int
	body   string
}

type fakeGemini struct {
	mutex    sync.Mutex
	replies  []scriptedReply
	requests []map[string]any
	paths    []string
	keys     []string
}

func (server *fakeGemini) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	body, _ := io.ReadAll(request.Body)

	var decoded map[string]any

	_ = json.Unmarshal(body, &decoded)

	server.requests = append(server.requests, decoded)
	server.paths = append(server.paths, request.URL.Path)
	server.keys = append(server.keys, request.Header.Get("x-goog-api-key"))

	reply := server.replies[0]
	if len(server.replies) > 1 {
		server.replies = server.replies[1:]
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(reply.status)
	_, _ = response.Write([]byte(reply.body))
}

type fakeClock struct {
	now    time.Time
	sleeps []time.Duration
}

func (clock *fakeClock) current() time.Time {
	return clock.now
}

func (clock *fakeClock) sleep(_ context.Context, delay time.Duration) error {
	clock.sleeps = append(clock.sleeps, delay)
	clock.now = clock.now.Add(delay)

	return nil
}

func newTestGemini(t *testing.T, replies ...scriptedReply) (*Gemini, *fakeGemini, *fakeClock) {
	t.Helper()

	server := &fakeGemini{replies: replies}
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)

	gemini, err := NewGemini(context.Background(), GeminiConfig{
		APIKey:          "test-key",
		Model:           testModel,
		MaxOutputTokens: 512,
		MinInterval:     6 * time.Second,
		RequestTimeout:  5 * time.Second,
		BaseURL:         httpServer.URL,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewGemini() error = %v", err)
	}

	clock := &fakeClock{now: time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)}
	gemini.now = clock.current
	gemini.throttle = newThrottle(6*time.Second, clock.current, clock.sleep)
	gemini.retries = defaultRetryPolicy(clock.sleep)

	return gemini, server, clock
}

func textReply(text string) scriptedReply {
	return scriptedReply{status: http.StatusOK, body: `{"candidates":[{"content":{"role":"model","parts":[{"text":"` +
		text + `"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`}
}

func rateLimitReply(details string) scriptedReply {
	return scriptedReply{status: http.StatusTooManyRequests, body: `{"error":{"code":429,"message":"quota exceeded",` +
		`"status":"RESOURCE_EXHAUSTED","details":[` + details + `]}}`}
}

func field(value any, path ...any) any {
	for _, step := range path {
		switch key := step.(type) {
		case string:
			object, _ := value.(map[string]any)
			value = object[key]
		case int:
			list, _ := value.([]any)
			if key >= len(list) {
				return nil
			}

			value = list[key]
		}
	}

	return value
}

func TestNewGeminiRequiresAPIKey(t *testing.T) {
	if _, err := NewGemini(
		context.Background(),
		GeminiConfig{Model: testModel},
		slog.New(slog.DiscardHandler),
	); !errors.Is(
		err,
		ErrMissingAPIKey,
	) {
		t.Fatalf("NewGemini() error = %v", err)
	}
}

func TestGeminiSendsSystemPromptToolsAndLimits(t *testing.T) {
	gemini, server, _ := newTestGemini(t, textReply("Готово"))

	_, err := gemini.Generate(context.Background(), Request{
		System:        "Ти помічник",
		History:       []Message{UserMessage("Покажи мої ділянки")},
		ThinkingLevel: ThinkingMinimal,
		Tools: []ToolDeclaration{{
			Name:        "list_plots",
			Description: "List plots",
			Parameters:  map[string]any{"type": "object", "additionalProperties": false},
		}},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	request := server.requests[0]

	checks := map[string][2]any{
		"path":           {server.paths[0], "/v1beta/models/" + testModel + ":generateContent"},
		"api key":        {server.keys[0], "test-key"},
		"system":         {field(request, "systemInstruction", "parts", 0, "text"), "Ти помічник"},
		"user text":      {field(request, "contents", 0, "parts", 0, "text"), "Покажи мої ділянки"},
		"max tokens":     {field(request, "generationConfig", "maxOutputTokens"), 512.0},
		"thinking level": {field(request, "generationConfig", "thinkingConfig", "thinkingLevel"), "MINIMAL"},
		"tool name":      {field(request, "tools", 0, "functionDeclarations", 0, "name"), "list_plots"},
		"tool schema": {
			field(request, "tools", 0, "functionDeclarations", 0, "parametersJsonSchema", "type"),
			"object",
		},
		"mode": {field(request, "toolConfig", "functionCallingConfig", "mode"), "VALIDATED"},
		"allowed": {
			field(request, "toolConfig", "functionCallingConfig", "allowedFunctionNames", 0),
			"list_plots",
		},
	}
	for name, check := range checks {
		if check[0] != check[1] {
			t.Errorf("%s = %#v, want %#v", name, check[0], check[1])
		}
	}
}

func TestGeminiReturnsToolCallsUsageAndKeepsThoughtSignature(t *testing.T) {
	gemini, server, _ := newTestGemini(t,
		scriptedReply{status: http.StatusOK, body: `{"candidates":[{"content":{"role":"model","parts":[` +
			`{"functionCall":{"id":"call-1","name":"list_plots","args":{"name_contains":"Дача"}},"thoughtSignature":"` +
			testSignature + `"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":100,` +
			`"toolUsePromptTokenCount":4,"candidatesTokenCount":20,"thoughtsTokenCount":30,"totalTokenCount":154}}`},
		textReply("Знайдено одну ділянку"),
	)

	first, err := gemini.Generate(context.Background(), Request{History: []Message{UserMessage("Знайди дачу")}})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	call := first.Message.ToolCalls[0]
	if call.ID != "call-1" || call.Name != "list_plots" || string(call.Arguments) != `{"name_contains":"Дача"}` {
		t.Fatalf("tool call = %#v", call)
	}

	if first.Usage != (Usage{InputTokens: 104, OutputTokens: 20, ThinkingTokens: 30, TotalTokens: 154}) {
		t.Fatalf("usage = %#v", first.Usage)
	}

	history := []Message{
		UserMessage("Знайди дачу"),
		first.Message,
		ToolResultsMessage(ToolResult{CallID: "call-1", Name: "list_plots", Content: `<tool_result>{}</tool_result>`}),
	}
	if _, err := gemini.Generate(context.Background(), Request{History: history}); err != nil {
		t.Fatalf("second Generate() error = %v", err)
	}

	second := server.requests[1]
	if signature := field(second, "contents", 1, "parts", 0, "thoughtSignature"); signature != testSignature {
		t.Fatalf("thought signature = %#v", signature)
	}

	response := field(second, "contents", 2, "parts", 0, "functionResponse")
	if field(response, "id") != "call-1" || field(response, "name") != "list_plots" ||
		field(response, "response", "output") != `<tool_result>{}</tool_result>` {
		t.Fatalf("function response = %#v", response)
	}
}

func TestGeminiSendsToolErrorsUnderErrorKey(t *testing.T) {
	gemini, server, _ := newTestGemini(t, textReply("Помилка валідації"))
	history := []Message{
		UserMessage("Створи ділянку"),
		{
			Role:      RoleModel,
			ToolCalls: []ToolCall{{ID: "c", Name: "create_plot", Arguments: json.RawMessage(`{"name":"x"}`)}},
		},
		ToolResultsMessage(ToolResult{CallID: "c", Name: "create_plot", Content: "invalid", IsError: true}),
	}

	if _, err := gemini.Generate(context.Background(), Request{History: history}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	request := server.requests[0]
	if field(request, "contents", 1, "parts", 0, "functionCall", "args", "name") != "x" ||
		field(request, "contents", 2, "parts", 0, "functionResponse", "response", "error") != "invalid" {
		t.Fatalf("contents = %#v", request["contents"])
	}
}

func TestGeminiWaitsForSuggestedRetryDelay(t *testing.T) {
	gemini, server, clock := newTestGemini(t,
		rateLimitReply(`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{"quotaId":`+
			`"GenerateRequestsPerMinutePerProjectPerModel-FreeTier"}]},`+
			`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"7s"}`),
		textReply("Готово"),
	)

	response, err := gemini.Generate(context.Background(), Request{History: []Message{UserMessage("Привіт")}})
	if err != nil || response.Message.Text != "Готово" {
		t.Fatalf("Generate() = %#v, %v", response, err)
	}

	if len(server.requests) != 2 || len(clock.sleeps) != 1 || clock.sleeps[0] != 7*time.Second {
		t.Fatalf("requests = %d, sleeps = %v", len(server.requests), clock.sleeps)
	}
}

func TestGeminiBacksOffThreeTimesAndGivesUp(t *testing.T) {
	gemini, server, clock := newTestGemini(t, rateLimitReply(""))

	_, err := gemini.Generate(context.Background(), Request{History: []Message{UserMessage("Привіт")}})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("Generate() error = %v", err)
	}

	want := []time.Duration{2 * time.Second, 4 * time.Second, 4 * time.Second, 2 * time.Second, 8 * time.Second}
	if len(server.requests) != 4 || !equalDurations(clock.sleeps, want) {
		t.Fatalf("requests = %d, sleeps = %v, want %v", len(server.requests), clock.sleeps, want)
	}
}

func TestGeminiStopsAtDailyQuotaWithoutRetrying(t *testing.T) {
	gemini, server, clock := newTestGemini(t, rateLimitReply(
		`{"@type":"type.googleapis.com/google.rpc.QuotaFailure","violations":[{"quotaMetric":`+
			`"generativelanguage.googleapis.com/generate_content_free_tier_requests","quotaId":`+
			`"GenerateRequestsPerDayPerProjectPerModel-FreeTier"}]},`+
			`{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"20s"}`,
	))

	_, err := gemini.Generate(context.Background(), Request{History: []Message{UserMessage("Привіт")}})
	if !errors.Is(err, ErrDailyQuotaExhausted) || len(server.requests) != 1 || len(clock.sleeps) != 0 {
		t.Fatalf("error = %v, requests = %d, sleeps = %v", err, len(server.requests), clock.sleeps)
	}
}

func TestGeminiDoesNotRetryBadRequest(t *testing.T) {
	gemini, server, _ := newTestGemini(t, scriptedReply{
		status: http.StatusBadRequest,
		body:   `{"error":{"code":400,"message":"invalid schema","status":"INVALID_ARGUMENT"}}`,
	})

	_, err := gemini.Generate(context.Background(), Request{History: []Message{UserMessage("Привіт")}})
	if err == nil || errors.Is(err, ErrRateLimited) || len(server.requests) != 1 {
		t.Fatalf("error = %v, requests = %d", err, len(server.requests))
	}
}

func TestGeminiReportsTruncatedEmptyAnswer(t *testing.T) {
	gemini, _, _ := newTestGemini(t, scriptedReply{status: http.StatusOK, body: `{"candidates":[{"content":` +
		`{"role":"model","parts":[]},"finishReason":"MAX_TOKENS"}],"usageMetadata":{"thoughtsTokenCount":512,` +
		`"totalTokenCount":600}}`})

	response, err := gemini.Generate(context.Background(), Request{History: []Message{UserMessage("Привіт")}})
	if !errors.Is(err, ErrOutputTruncated) || response.Usage.TotalTokens != 600 {
		t.Fatalf("Generate() = %#v, %v", response, err)
	}
}

func TestGeminiReportsBlockedPrompt(t *testing.T) {
	gemini, _, _ := newTestGemini(t, scriptedReply{
		status: http.StatusOK,
		body:   `{"promptFeedback":{"blockReason":"SAFETY"},"usageMetadata":{"totalTokenCount":3}}`,
	})

	_, err := gemini.Generate(context.Background(), Request{History: []Message{UserMessage("Привіт")}})
	if !errors.Is(err, ErrResponseBlocked) || !strings.Contains(err.Error(), "SAFETY") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGeminiKeepsMinimumIntervalBetweenCalls(t *testing.T) {
	gemini, _, clock := newTestGemini(t, textReply("Готово"))

	for range 2 {
		if _, err := gemini.Generate(
			context.Background(),
			Request{History: []Message{UserMessage("Привіт")}},
		); err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
	}

	if len(clock.sleeps) != 1 || clock.sleeps[0] != 6*time.Second {
		t.Fatalf("sleeps = %v", clock.sleeps)
	}
}

func equalDurations(got, want []time.Duration) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
