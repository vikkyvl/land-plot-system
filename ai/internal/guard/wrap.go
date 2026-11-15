package guard

import (
	"regexp"
	"strings"
)

const (
	toolResultOpening = `<tool_result name="`
	toolResultClosing = "</tool_result>"
	escapedTagStart   = "&lt;"
)

var toolResultTag = regexp.MustCompile(`(?i)<\s*/?\s*tool_result`)

func WrapToolResult(toolName, text string) string {
	escaped := toolResultTag.ReplaceAllStringFunc(text, func(tag string) string {
		return escapedTagStart + strings.TrimPrefix(tag, "<")
	})

	return toolResultOpening + toolName + `">` + escaped + toolResultClosing
}
