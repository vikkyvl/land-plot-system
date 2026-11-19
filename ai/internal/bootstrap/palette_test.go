package bootstrap

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/guard"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
)

var ansiSequence = regexp.MustCompile("\x1b\\[[0-9;]*m")

func coloredPalette(output *bytes.Buffer) palette {
	renderer := lipgloss.NewRenderer(output)
	renderer.SetColorProfile(termenv.ANSI256)

	return newPaletteWith(true, renderer)
}

func writeSession(printer *console) {
	printer.header(agent.ModeNonReasoning, "gemini-3.5-flash-lite", "Покажи мої ділянки")
	printer.step(
		agent.Step{Kind: agent.StepModel, Usage: llm.Usage{InputTokens: 800, OutputTokens: 20, TotalTokens: 820}},
	)
	printer.step(
		agent.Step{
			Kind:      agent.StepTool,
			ToolName:  "list_plots",
			Arguments: json.RawMessage(`{}`),
			Output:    `{"count":0}`,
		},
	)
	printer.step(agent.Step{Kind: agent.StepTool, ToolName: "create_plot", Output: "invalid", IsError: true})
	printer.result(agent.Result{
		Answer:     guard.MessageBudgetExhausted,
		Steps:      1,
		Usage:      llm.Usage{InputTokens: 800, OutputTokens: 20, TotalTokens: 820},
		StopReason: guard.ErrTokenBudgetExhausted,
	})
}

func TestColorsAreDisabledOutsideATerminal(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "output.txt"))
	if err != nil {
		t.Fatalf("create file error = %v", err)
	}
	defer file.Close()

	if colorsEnabled(&bytes.Buffer{}) || colorsEnabled(file) {
		t.Fatal("colors are enabled for a buffer or a regular file")
	}

	t.Setenv("NO_COLOR", "1")

	if colorsEnabled(os.Stdout) {
		t.Fatal("colors are enabled although NO_COLOR is set")
	}
}

func TestPlainConsoleOutputIsUnchanged(t *testing.T) {
	var output bytes.Buffer

	writeSession(newConsole(&output))

	want := "Режим: non-reasoning, модель: gemini-3.5-flash-lite\nЗапит: Покажи мої ділянки\n\n" +
		"[модель 1] токени: вхід 800, вихід 20, роздуми 0, усього 820\n" +
		"  → list_plots {}\n" +
		"  ← {\"count\":0}\n" +
		"  → create_plot \n" +
		"  ← помилка: invalid\n" +
		"\nВідповідь:\nБюджет токенів вичерпано\n" +
		"Причина зупинки: бюджет токенів вичерпано\n" +
		"\nКроків з інструментами: 1. Викликів моделі: 1. Токенів: 820 (вхід 800, вихід 20, роздуми 0).\n"
	if output.String() != want {
		t.Fatalf("output =\n%q\nwant\n%q", output.String(), want)
	}
}

func TestColoredConsoleKeepsTheLinesAndFramesTheAnswer(t *testing.T) {
	var output bytes.Buffer

	printer := newConsole(&output)
	printer.colors = coloredPalette(&output)
	writeSession(printer)

	colored := output.String()
	plain := ansiSequence.ReplaceAllString(colored, "")

	if !ansiSequence.MatchString(colored) {
		t.Fatalf("output has no colors: %q", colored)
	}

	for _, want := range []string{
		"[модель 1] токени: вхід 800, вихід 20, роздуми 0, усього 820\n",
		"  → list_plots {}\n",
		"  ← {\"count\":0}\n",
		"  ← помилка: invalid\n",
		"Причина зупинки: бюджет токенів вичерпано\n",
		"\nКроків з інструментами: 1. Викликів моделі: 1. Токенів: 820 (вхід 800, вихід 20, роздуми 0).\n",
		"╭",
		"│ " + answerTitle,
		"│ Бюджет токенів вичерпано",
		"╰",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("plain text does not contain %q:\n%s", want, plain)
		}
	}
}

func TestPaintKeepsSurroundingLineBreaks(t *testing.T) {
	var output bytes.Buffer

	colors := coloredPalette(&output)

	painted := colors.paint(colors.bold, "\n\nПідсумок\n")
	if !strings.HasPrefix(painted, "\n\n") || !strings.HasSuffix(painted, "\n") ||
		ansiSequence.ReplaceAllString(painted, "") != "\n\nПідсумок\n" {
		t.Fatalf("paint() = %q", painted)
	}

	if plain := newPaletteWith(false, lipgloss.NewRenderer(&output)).paint(colors.bold, "текст\n"); plain != "текст\n" {
		t.Fatalf("disabled paint() = %q", plain)
	}
}

func TestTaskStatesHaveTheirColors(t *testing.T) {
	var output bytes.Buffer

	colors := coloredPalette(&output)
	for state, want := range map[a2a.TaskState]lipgloss.Style{
		a2a.TaskStateWorking:   colors.warning,
		a2a.TaskStateCompleted: colors.success,
		a2a.TaskStateFailed:    colors.failure,
		a2a.TaskStateSubmitted: colors.muted,
	} {
		if colors.taskState(state).Render("x") != want.Render("x") {
			t.Fatalf("%s has a wrong color", state)
		}
	}

	printer := taskPrinter{output: &output, colors: colors}
	printer.event(a2a.NewStatusUpdateEvent(a2a.TaskInfo{TaskID: "t", ContextID: "c"}, a2a.TaskStateCompleted, nil))

	if plain := ansiSequence.ReplaceAllString(output.String(), ""); plain != "[TASK_STATE_COMPLETED] \n" ||
		!ansiSequence.MatchString(output.String()) {
		t.Fatalf("status line = %q", output.String())
	}
}
