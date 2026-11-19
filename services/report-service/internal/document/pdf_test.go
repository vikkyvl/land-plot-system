package document

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

var testCreatedAt = time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)

func carpathianReport() domain.Report {
	worstSide := 1

	return domain.Report{
		ID: "3c51abb9-6632-4d7d-9795-000228d15d74", PlotID: "plot-1", PlotName: "Полонина над Ворохтою", PlotVersion: 2,
		AreaM2: 198601, AreaAres: 1986.01, AreaHectares: 19.8601, PerimeterM: 1783, CreatedAt: testCreatedAt,
		Accuracy: &domain.AreaAccuracy{
			AreaMarginM2: 8500, AreaMarginHectares: 0.85, Percent: 4.3, Confidence: 0.95,
			ScaleShare: 0.9, ObstructionShare: 0.1, WorstSide: &worstSide, GroundShiftM: 4, Basis: "capture",
		},
		Terrain: &domain.Terrain{
			Status: domain.DataAvailable, SlopeDegrees: 16.74, SurfaceAreaM2: 207381,
			SurfaceIncreasePercent: 4.42, MinElevationM: 1301, MaxElevationM: 1532, ReliefM: 231,
		},
		Climate: &domain.Climate{
			Status: domain.DataAvailable, FirstYear: 2016, LastYear: 2025, GrowingDays: 180.4,
			LastSpringFrost:  &domain.MonthDay{Month: time.May, Day: 13},
			FirstAutumnFrost: &domain.MonthDay{Month: time.October, Day: 3},
			Seasons: []domain.SeasonClimate{
				{
					Season:          domain.SeasonWinter,
					MeanC:           -4.2,
					MeanMinC:        -7.3,
					MeanMaxC:        -1.2,
					PrecipitationMm: 300,
					FrostDays:       85.2,
				},
			},
		},
		Boundary: []domain.Coordinate{
			{Latitude: 48.14, Longitude: 24.48},
			{Latitude: 48.14, Longitude: 24.486},
			{Latitude: 48.144, Longitude: 24.486},
			{Latitude: 48.144, Longitude: 24.48},
		},
	}
}

func drawnText(t *testing.T, report domain.Report, language domain.Language) (string, []byte) {
	t.Helper()

	content, drawn, err := render(report, language)
	if err != nil {
		t.Fatalf("render() error = %v", err)
	}

	return strings.Join(drawn, "\n"), content
}

func TestPDFShowsThePlotTheAreaAndTheSources(t *testing.T) {
	text, content := drawnText(t, carpathianReport(), domain.LanguageUkrainian)

	if !bytes.HasPrefix(content, []byte("%PDF-")) || !bytes.Contains(content, []byte("/FontFile2")) {
		t.Fatal("content is not a PDF with an embedded TrueType font")
	}

	for _, expected := range []string{
		"Звіт · Полонина над Ворохтою",
		"Звіт № 3C51 · сформовано 18.11.2026",
		"19,86 га ± 0,85 га",
		"периметр: 1 783 м",
		"Середній ухил 16,7°",
		"Зима", "−4,2 °C", "300 мм",
		"Останній весняний заморозок: 13 травня",
		"180 днів із середньою температурою ≥ 5 °C",
		"48.14000",
		"Дані: Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5, Copernicus DEM GLO-90",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("PDF text lacks %q", expected)
		}
	}
}

func TestPDFInEnglish(t *testing.T) {
	text, _ := drawnText(t, carpathianReport(), domain.LanguageEnglish)

	for _, expected := range []string{
		"Report No. 3C51 · created 18/11/2026", "19.86 ha ± 0.85 ha", "perimeter: 1,783 m",
		"Last spring frost: 13 May", "180 days with a mean temperature ≥ 5 °C",
		"Data: Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5, Copernicus DEM GLO-90",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("PDF text lacks %q", expected)
		}
	}
}

func TestPDFOfAnOlderOrUnavailableReportSaysThereIsNoData(t *testing.T) {
	older := carpathianReport()
	older.Accuracy, older.Terrain, older.Boundary = nil, nil, nil
	older.Climate = &domain.Climate{Status: domain.DataUnavailable}
	deletedAt := testCreatedAt
	older.PlotDeletedAt = &deletedAt

	text, _ := drawnText(t, older, domain.LanguageUkrainian)

	if strings.Count(text, "Даних немає") != 4 || !strings.Contains(text, "Похибка: даних немає") ||
		!strings.Contains(text, "Ділянку видалено") || !strings.Contains(text, "19,86 га") {
		t.Fatalf("text = %s", text)
	}
}

func TestPDFMarksAnAssumedMarginAndUsesUkrainianPlurals(t *testing.T) {
	report := carpathianReport()
	report.Accuracy.Basis = "assumed"
	report.Climate.GrowingDays = 231

	text, _ := drawnText(t, report, domain.LanguageUkrainian)

	if !strings.Contains(text, "≈ ± 0,85 га") || !strings.Contains(text, "Орієнтовно") ||
		!strings.Contains(text, "231 день із середньою") {
		t.Fatalf("text = %s", text)
	}
}

func TestGeoJSONComesFromTheStoredBoundary(t *testing.T) {
	content, err := Renderer{}.GeoJSON(carpathianReport())
	if err != nil {
		t.Fatalf("GeoJSON() error = %v", err)
	}

	var collection geoJSONCollection
	if err := json.Unmarshal(content, &collection); err != nil {
		t.Fatalf("decode: %v", err)
	}

	ring := collection.Features[0].Geometry.Coordinates[0]
	if len(ring) != 5 || ring[0] != ring[4] || ring[1] != [2]float64{24.486, 48.14} ||
		collection.Features[0].Properties.ReportNumber != "3C51" {
		t.Fatalf("collection = %+v", collection)
	}

	older := carpathianReport()
	older.Boundary = nil

	if _, err := (Renderer{}).GeoJSON(older); !errors.Is(err, domain.ErrBoundaryNotRecorded) {
		t.Fatalf("error = %v, want ErrBoundaryNotRecorded", err)
	}
}
