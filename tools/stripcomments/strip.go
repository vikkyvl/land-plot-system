package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"strings"
)

var directivePrefixes = []string{"//go:", "//nolint", "//lint:", "//line "}

type span struct {
	start int
	end   int
}

func stripComments(filename string, source []byte) ([]byte, int, error) {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, filename, source, parser.ParseComments)
	if err != nil {
		return nil, 0, err
	}

	spans := removableSpans(fileSet, file, source)
	if len(spans) == 0 {
		return source, 0, nil
	}

	stripped, err := format.Source(removeSpans(source, spans))
	if err != nil {
		return nil, 0, err
	}

	return stripped, len(spans), nil
}

func removableSpans(fileSet *token.FileSet, file *ast.File, source []byte) []span {
	var spans []span

	for _, group := range file.Comments {
		for _, comment := range group.List {
			if isDirective(comment.Text) {
				continue
			}

			spans = append(spans, commentSpan(fileSet, comment, source))
		}
	}

	return spans
}

func isDirective(text string) bool {
	return slices.ContainsFunc(directivePrefixes, func(prefix string) bool {
		return strings.HasPrefix(text, prefix)
	})
}

func commentSpan(fileSet *token.FileSet, comment *ast.Comment, source []byte) span {
	start := fileSet.Position(comment.Pos()).Offset
	end := fileSet.Position(comment.End()).Offset

	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	lineEnd := len(source)

	if index := bytes.IndexByte(source[end:], '\n'); index >= 0 {
		lineEnd = end + index
	}

	before := bytes.TrimSpace(source[lineStart:start])
	after := bytes.TrimSpace(source[end:lineEnd])

	if len(before) == 0 && len(after) == 0 {
		return span{start: lineStart, end: min(lineEnd+1, len(source))}
	}

	for start > lineStart && (source[start-1] == ' ' || source[start-1] == '\t') {
		start--
	}

	return span{start: start, end: end}
}

func removeSpans(source []byte, spans []span) []byte {
	slices.SortFunc(spans, func(first, second span) int { return first.start - second.start })

	var result bytes.Buffer

	position := 0

	for _, removed := range spans {
		if removed.start < position {
			continue
		}

		result.Write(source[position:removed.start])

		position = removed.end
	}

	result.Write(source[position:])

	return result.Bytes()
}
