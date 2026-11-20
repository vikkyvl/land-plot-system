package domain

import (
	"strings"
	"unicode"
)

type Language string

const (
	LanguageUkrainian Language = "uk"
	LanguageEnglish   Language = "en"
)

const (
	reportNumberLength = 4
	fileNameSeparator  = "-"
	maximumNameRunes   = 60
)

type ReportFile struct {
	Name        string
	ContentType string
	Content     []byte
}

const (
	ContentTypePDF     = "application/pdf"
	ContentTypeGeoJSON = "application/geo+json"
	ExtensionPDF       = "pdf"
	ExtensionGeoJSON   = "geojson"
)

var reportFilePrefixes = map[Language]string{LanguageUkrainian: "Звіт", LanguageEnglish: "Report"}

func (language Language) ReportFilePrefix() string {
	return reportFilePrefixes[language]
}

func ParseLanguage(value string) (Language, error) {
	switch Language(value) {
	case "", LanguageUkrainian:
		return LanguageUkrainian, nil
	case LanguageEnglish:
		return LanguageEnglish, nil
	default:
		return "", ErrInvalidLanguage
	}
}

func (report Report) Number() string {
	compact := strings.ReplaceAll(report.ID, "-", "")
	if len(compact) > reportNumberLength {
		compact = compact[:reportNumberLength]
	}

	return strings.ToUpper(compact)
}

func (report Report) FileName(prefix, extension string) string {
	return strings.Join(
		[]string{prefix, safeName(report.PlotName), report.Number()},
		fileNameSeparator,
	) + "." + extension
}

func safeName(name string) string {
	var builder strings.Builder

	separated := false
	written := 0

	for _, character := range name {
		if written >= maximumNameRunes {
			break
		}

		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			builder.WriteRune(character)

			separated = false
			written++

			continue
		}

		if !separated && builder.Len() > 0 {
			builder.WriteString(fileNameSeparator)

			separated = true
			written++
		}
	}

	return strings.TrimSuffix(builder.String(), fileNameSeparator)
}
