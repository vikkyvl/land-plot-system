package bootstrap

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/vleshchenko/land-plot-system/ai/internal/agent"
	"github.com/vleshchenko/land-plot-system/ai/internal/llm"
)

const previewRunes = 200

type console struct {
	output     io.Writer
	modelCalls int
}

func newConsole(output io.Writer) *console {
	return &console{output: output}
}

func (printer *console) header(mode agent.Mode, model, query string) {
	printer.printf(formatHeader, mode, model, query)
}

func (printer *console) step(step agent.Step) {
	switch step.Kind {
	case agent.StepModel:
		printer.modelCalls++
		printer.printf(formatModelStep, printer.modelCalls, step.Usage.InputTokens, step.Usage.OutputTokens,
			step.Usage.ThinkingTokens, step.Usage.TotalTokens)
	case agent.StepTool:
		printer.printf(formatToolCall, step.ToolName, compact(string(step.Arguments)))

		if step.IsError {
			printer.printf(formatToolError, preview(step.Output))
			return
		}

		printer.printf(formatToolResult, preview(step.Output))
	}
}

func (printer *console) result(result agent.Result) {
	printer.printf(formatAnswer, result.Answer)

	if result.StopReason != nil {
		printer.printf(formatStopReason, result.StopReason)
	}

	printer.summary(result)
}

func (printer *console) failure(result agent.Result, err error) {
	printer.printf(formatFailure, err)
	printer.summary(result)
}

func (printer *console) summary(result agent.Result) {
	printer.printf(formatSummary, result.Steps, printer.modelCalls, totals(result.Usage))
}

func (printer *console) printf(format string, arguments ...any) {
	_, _ = fmt.Fprintf(printer.output, format, arguments...)
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
