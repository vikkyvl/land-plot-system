package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/config"
	"github.com/vleshchenko/land-plot-system/ai/internal/guard"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
)

func TestParseFCOptionsReadsModeAndQuery(t *testing.T) {
	options, err := parseFCOptions([]string{"-mode", "reasoning", "-v", "Покажи", "мої", "ділянки"}, io.Discard)
	if err != nil {
		t.Fatalf("parseFCOptions() error = %v", err)
	}

	if options.mode != agent.ModeReasoning || !options.verbose || options.query != "Покажи мої ділянки" {
		t.Fatalf("options = %#v", options)
	}
}

func TestParseFCOptionsDefaultsToNonReasoning(t *testing.T) {
	options, err := parseFCOptions([]string{"Покажи мої ділянки"}, io.Discard)
	if err != nil || options.mode != agent.ModeNonReasoning || options.verbose {
		t.Fatalf("parseFCOptions() = %#v, %v", options, err)
	}
}

func TestParseFCOptionsRejectsInvalidArguments(t *testing.T) {
	for name, arguments := range map[string][]string{
		"missing query": {"-mode", "reasoning"},
		"blank query":   {"   "},
		"unknown mode":  {"-mode", "fast", "Покажи мої ділянки"},
		"unknown flag":  {"-interactive"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseFCOptions(arguments, io.Discard); err == nil {
				t.Fatal("parseFCOptions() succeeded")
			}
		})
	}
}

func TestRunFCClientPrintsUsageForInvalidArguments(t *testing.T) {
	var stderr bytes.Buffer

	if code := RunFCClient(context.Background(), nil, io.Discard, &stderr); code != ExitUsage ||
		!strings.Contains(stderr.String(), "usage: fc-client") {
		t.Fatalf("RunFCClient() = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunFCClientRequiresAnAPIKey(t *testing.T) {
	t.Setenv("AI_USER_EMAIL", "agent@example.com")
	t.Setenv("AI_USER_PASSWORD", "AgentPassword123")
	t.Setenv("LLM_API_KEY", "")

	var stderr bytes.Buffer

	code := RunFCClient(context.Background(), []string{"Покажи мої ділянки"}, io.Discard, &stderr)
	if code != ExitError || !strings.Contains(stderr.String(), llm.ErrMissingAPIKey.Error()) {
		t.Fatalf("RunFCClient() = %d, stderr = %q", code, stderr.String())
	}
}

func TestThinkingLevelFollowsTheMode(t *testing.T) {
	settings := config.LLM{ReasoningThinkingLevel: "medium"}

	if level := thinkingLevel(agent.ModeNonReasoning, settings); level != llm.ThinkingMinimal {
		t.Fatalf("non-reasoning level = %q", level)
	}

	if level := thinkingLevel(agent.ModeReasoning, settings); level != llm.ThinkingMedium {
		t.Fatalf("reasoning level = %q", level)
	}
}

func TestConsolePrintsStepsAnswerAndTotals(t *testing.T) {
	var output bytes.Buffer

	printer := newConsole(&output)
	printer.header(agent.ModeNonReasoning, "gemini-3.5-flash-lite", "Покажи мої ділянки")
	printer.step(
		agent.Step{Kind: agent.StepModel, Usage: llm.Usage{InputTokens: 800, OutputTokens: 20, TotalTokens: 820}},
	)
	printer.step(agent.Step{
		Kind:      agent.StepTool,
		ToolName:  "list_plots",
		Arguments: json.RawMessage(`{ "name_contains": "Дача" }`),
		Output:    `{"count":1,"plots":[{"name":"Дача"}]}`,
	})
	printer.step(agent.Step{Kind: agent.StepTool, ToolName: "create_plot", Output: "invalid", IsError: true})
	printer.result(agent.Result{
		Answer:     guard.MessageBudgetExhausted,
		Steps:      1,
		Usage:      llm.Usage{InputTokens: 800, OutputTokens: 20, TotalTokens: 820},
		StopReason: guard.ErrTokenBudgetExhausted,
	})

	text := output.String()
	for _, want := range []string{
		"Режим: non-reasoning, модель: gemini-3.5-flash-lite",
		"[модель 1] токени: вхід 800, вихід 20, роздуми 0, усього 820",
		`→ list_plots { "name_contains": "Дача" }`,
		`← {"count":1,"plots":[{"name":"Дача"}]}`,
		"← помилка: invalid",
		"Бюджет токенів вичерпано",
		"Причина зупинки: бюджет токенів вичерпано",
		"Кроків з інструментами: 1. Викликів моделі: 1. Токенів: 820 (вхід 800, вихід 20, роздуми 0).",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output does not contain %q:\n%s", want, text)
		}
	}
}

func TestConsoleShortensLongToolOutput(t *testing.T) {
	long := strings.Repeat("ї", previewRunes+50)

	if shortened := preview(long); !strings.HasSuffix(shortened, ellipsis) ||
		len([]rune(shortened)) != previewRunes+1 {
		t.Fatalf("preview() = %d runes", len([]rune(shortened)))
	}
}

func TestConsolePrintsFailures(t *testing.T) {
	var output bytes.Buffer

	newConsole(&output).failure(agent.Result{}, errors.New("денний ліміт безкоштовного тарифу вичерпано"))

	if !strings.Contains(output.String(), "Помилка: денний ліміт") {
		t.Fatalf("output = %q", output.String())
	}
}
