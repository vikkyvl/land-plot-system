package bootstrap

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
)

const previewRunes = 200

type console struct {
	output     io.Writer
	colors     palette
	modelCalls int
}

func newConsole(output io.Writer) *console {
	return &console{output: output, colors: newPalette(output)}
}

func (printer *console) header(mode agent.Mode, model, query string) {
	printer.write(fmt.Sprintf(formatHeader, mode, model, query))
}

func (printer *console) step(step agent.Step) {
	switch step.Kind {
	case agent.StepModel:
		printer.modelCalls++
		printer.styled(printer.colors.muted, formatModelStep, printer.modelCalls, step.Usage.InputTokens,
			step.Usage.OutputTokens, step.Usage.ThinkingTokens, step.Usage.TotalTokens)
	case agent.StepTool:
		printer.styled(printer.colors.tool, formatToolCall, step.ToolName, compact(string(step.Arguments)))

		if step.IsError {
			printer.styled(printer.colors.failure, formatToolError, preview(step.Output))
			return
		}

		printer.styled(printer.colors.muted, formatToolResult, preview(step.Output))
	}
}

func (printer *console) result(result agent.Result) {
	printer.answer(result.Answer)

	if result.StopReason != nil {
		printer.styled(printer.colors.failure, formatStopReason, result.StopReason)
	}

	printer.summary(result)
}

func (printer *console) failure(result agent.Result, err error) {
	printer.styled(printer.colors.failure, formatFailure, err)
	printer.summary(result)
}

func (printer *console) summary(result agent.Result) {
	printer.styled(printer.colors.bold, formatSummary, result.Steps, printer.modelCalls, totals(result.Usage))
}

func (printer *console) answer(answer string) {
	if !printer.colors.enabled {
		printer.write(fmt.Sprintf(formatAnswer, answer))
		return
	}

	printer.write(lineBreak + printer.colors.answerBox(answer) + lineBreak)
}

func (printer *console) styled(style lipgloss.Style, format string, arguments ...any) {
	printer.write(printer.colors.paint(style, fmt.Sprintf(format, arguments...)))
}

func (printer *console) write(text string) {
	_, _ = io.WriteString(printer.output, text)
}

func totals(usage llm.Usage) string {
	return fmt.Sprintf(
		formatUsageTotals,
		usage.TotalTokens,
		usage.InputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
	)
}

func compact(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func preview(text string) string {
	text = compact(text)
	if utf8.RuneCountInString(text) <= previewRunes {
		return text
	}

	return string([]rune(text)[:previewRunes]) + ellipsis
}
