package bootstrap

import (
	"io"
	"os"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/charmbracelet/lipgloss"
)

const (
	envNoColor      = "NO_COLOR"
	colorMuted      = lipgloss.Color("244")
	colorTool       = lipgloss.Color("39")
	colorFailure    = lipgloss.Color("196")
	colorWarning    = lipgloss.Color("214")
	colorSuccess    = lipgloss.Color("42")
	colorAnswerEdge = lipgloss.Color("63")
	answerTitle     = "Відповідь"
	answerPadding   = 1
	lineBreak       = "\n"
)

type palette struct {
	enabled bool
	muted   lipgloss.Style
	tool    lipgloss.Style
	failure lipgloss.Style
	warning lipgloss.Style
	success lipgloss.Style
	bold    lipgloss.Style
	answer  lipgloss.Style
	title   lipgloss.Style
}

func newPalette(output io.Writer) palette {
	return newPaletteWith(colorsEnabled(output), lipgloss.NewRenderer(output))
}

func newPaletteWith(enabled bool, renderer *lipgloss.Renderer) palette {
	return palette{
		enabled: enabled,
		muted:   renderer.NewStyle().Foreground(colorMuted),
		tool:    renderer.NewStyle().Foreground(colorTool),
		failure: renderer.NewStyle().Foreground(colorFailure),
		warning: renderer.NewStyle().Foreground(colorWarning),
		success: renderer.NewStyle().Foreground(colorSuccess),
		bold:    renderer.NewStyle().Bold(true),
		answer: renderer.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAnswerEdge).
			Padding(0, answerPadding),
		title: renderer.NewStyle().Bold(true).Foreground(colorAnswerEdge),
	}
}

func colorsEnabled(output io.Writer) bool {
	if _, disabled := os.LookupEnv(envNoColor); disabled {
		return false
	}

	file, ok := output.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()

	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (colors palette) paint(style lipgloss.Style, text string) string {
	if !colors.enabled {
		return text
	}

	body := strings.TrimLeft(text, lineBreak)
	leading := text[:len(text)-len(body)]
	trimmed := strings.TrimRight(body, lineBreak)
	trailing := body[len(trimmed):]

	return leading + style.Render(trimmed) + trailing
}

func (colors palette) answerBox(answer string) string {
	return colors.answer.Render(colors.title.Render(answerTitle) + lineBreak + answer)
}

func (colors palette) taskState(state a2a.TaskState) lipgloss.Style {
	switch state {
	case a2a.TaskStateWorking:
		return colors.warning
	case a2a.TaskStateCompleted:
		return colors.success
	case a2a.TaskStateFailed, a2a.TaskStateRejected, a2a.TaskStateCanceled:
		return colors.failure
	default:
		return colors.muted
	}
}
