package bootstrap

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/ai/internal/a2aagent"
	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/tools"
)

const testCallPlot = `{"id":"4b6f1a52-38f5-4c3e-9e8b-1d2f3a4b5c6d","name":"Сад","status":"CONFIRMED","version":1}`

type scriptedRunner struct {
	outcome string
}

func (runner scriptedRunner) Run(_ context.Context, _ string, observe agent.Observer) (agent.Result, error) {
	observe(agent.Step{Kind: agent.StepTool, ToolName: tools.CreatePlot, Output: testCallPlot})
	observe(agent.Step{
		Kind:     agent.StepTool,
		ToolName: tools.WaitPlotConfirmed,
		Output:   `{"outcome":"` + runner.outcome + `","plot":` + testCallPlot + `}`,
	})

	return agent.Result{Answer: "Готово: " + runner.outcome, Steps: 2}, nil
}

func startProducer(t *testing.T, outcome string) string {
	t.Helper()

	executor := a2aagent.NewExecutor(
		scriptedRunner{outcome: outcome},
		a2aagent.DecideProducer,
		slog.New(slog.DiscardHandler),
	)

	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		a2aagent.NewHandler(a2aagent.ProducerCard(server.URL), executor).ServeHTTP(response, request)
	}))
	t.Cleanup(server.Close)

	return server.URL
}

func TestParseA2ACallOptions(t *testing.T) {
	options, err := parseA2ACallOptions(
		[]string{"-timeout", "30s", "http://localhost:8092/", "Створи", "ділянку"},
		io.Discard,
	)
	if err != nil {
		t.Fatalf("parseA2ACallOptions() error = %v", err)
	}

	if options.agentURL != "http://localhost:8092" || options.text != "Створи ділянку" ||
		options.timeout != 30*time.Second {
		t.Fatalf("options = %#v", options)
	}

	for name, arguments := range map[string][]string{
		"no arguments":  nil,
		"no text":       {"http://localhost:8092"},
		"relative URL":  {"localhost:8092", "Створи"},
		"zero timeout":  {"-timeout", "0s", "http://localhost:8092", "Створи"},
		"blank text":    {"http://localhost:8092", "  "},
		"file scheme":   {"file:///etc/passwd", "Створи"},
		"unknown flag":  {"-stream", "http://localhost:8092", "Створи"},
		"missing value": {"-timeout"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseA2ACallOptions(arguments, io.Discard); err == nil {
				t.Fatal("parseA2ACallOptions() succeeded")
			}
		})
	}
}

func TestRunA2ACallPrintsStatesAndArtifact(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := RunA2ACall(context.Background(), []string{startProducer(t, tools.OutcomeConfirmed), "Створи ділянку Сад"},
		&stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("RunA2ACall() = %d, stderr = %q, stdout = %q", code, stderr.String(), stdout.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"Агент: Land plot producer agent 1.0.0, skills: create_plot, update_plot",
		"Задача: Створи ділянку Сад",
		"TASK_STATE_SUBMITTED",
		"[TASK_STATE_WORKING]",
		`[артефакт plot] {"name":"Сад","outcome":"confirmed","plot_id":"4b6f1a52-38f5-4c3e-9e8b-1d2f3a4b5c6d"`,
		"[TASK_STATE_COMPLETED] Готово: confirmed",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
}

func TestRunA2ACallReturnsErrorForFailedTask(t *testing.T) {
	var stdout bytes.Buffer

	code := RunA2ACall(context.Background(), []string{startProducer(t, tools.OutcomeCancelled), "Створи ділянку FAIL"},
		&stdout, io.Discard)
	if code != ExitError ||
		!strings.Contains(stdout.String(), "[TASK_STATE_FAILED] Сага завершилась без підтвердження: cancelled") ||
		strings.Contains(stdout.String(), "[артефакт") {
		t.Fatalf("RunA2ACall() = %d, stdout = %q", code, stdout.String())
	}
}

func TestRunA2ACallReportsUnreachableAgent(t *testing.T) {
	var stderr bytes.Buffer

	if code := RunA2ACall(context.Background(), []string{"-timeout", "2s", "http://127.0.0.1:1", "Створи"}, io.Discard,
		&stderr); code != ExitError || stderr.Len() == 0 {
		t.Fatalf("RunA2ACall() = %d, stderr = %q", code, stderr.String())
	}
}
