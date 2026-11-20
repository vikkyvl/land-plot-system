package domain

import (
	"errors"
	"testing"
)

func TestReportFileNameKeepsOnlySafeCharacters(t *testing.T) {
	report := Report{ID: "3c51abb9-6632-4d7d-9795-000228d15d74", PlotName: `Поле "біля" ставка / №2: <нове>`}

	if name := report.FileName(
		LanguageUkrainian.ReportFilePrefix(),
		ExtensionPDF,
	); name != "Звіт-Поле-біля-ставка-2-нове-3C51.pdf" {
		t.Fatalf("FileName() = %q", name)
	}

	if name := report.FileName(
		LanguageEnglish.ReportFilePrefix(),
		ExtensionGeoJSON,
	); name != "Report-Поле-біля-ставка-2-нове-3C51.geojson" {
		t.Fatalf("FileName() = %q", name)
	}
}

func TestParseLanguage(t *testing.T) {
	for value, want := range map[string]Language{"": LanguageUkrainian, "uk": LanguageUkrainian, "en": LanguageEnglish} {
		if got, err := ParseLanguage(value); err != nil || got != want {
			t.Errorf("ParseLanguage(%q) = %q, %v", value, got, err)
		}
	}

	if _, err := ParseLanguage("de"); !errors.Is(err, ErrInvalidLanguage) {
		t.Fatalf("ParseLanguage(de) error = %v", err)
	}
}
