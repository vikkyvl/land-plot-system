package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const exportReportID = "3c51abb9-6632-4d7d-9795-000228d15d74"

func exportReport() domain.Report {
	return domain.Report{
		ID: exportReportID, OwnerID: testOwnerID, PlotID: "plot-1", PlotName: "Поле біля ставка", PlotVersion: 1,
		AreaM2: 24800, AreaAres: 248, AreaHectares: 2.48, PerimeterM: 630,
		CreatedAt: time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC),
		Boundary: []domain.Coordinate{
			{
				Latitude:  50.45,
				Longitude: 30.52,
			},
			{Latitude: 50.45, Longitude: 30.523},
			{Latitude: 50.452, Longitude: 30.523},
		},
	}
}

func TestReportPDFIsAnAttachmentWithASafeName(t *testing.T) {
	response := serve(newTestRouter(&reportServiceStub{report: exportReport()}),
		authorizedRequest(http.MethodGet, "/api/v1/reports/"+exportReportID+"/pdf?lang=uk", ""))

	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/pdf" ||
		!bytes.HasPrefix(response.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("status = %d headers = %v", response.Code, response.Header())
	}

	disposition := response.Header().Get("Content-Disposition")
	if !strings.HasPrefix(disposition, `attachment; filename="____-____-____-______-3C51.pdf"`) ||
		!strings.Contains(disposition, "filename*=UTF-8''%D0%97%D0%B2%D1%96%D1%82-") ||
		response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("Content-Disposition = %q", disposition)
	}
}

func TestReportPDFOfADeletedPlotIsStillBuilt(t *testing.T) {
	report := exportReport()
	deletedAt := report.CreatedAt
	report.PlotDeletedAt = &deletedAt

	response := serve(newTestRouter(&reportServiceStub{report: report}),
		authorizedRequest(http.MethodGet, "/api/v1/reports/"+exportReportID+"/pdf?lang=en", ""))

	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Disposition"), "Report-") {
		t.Fatalf("status = %d headers = %v", response.Code, response.Header())
	}
}

func TestReportExportsAreRefusedForAnotherOwnerAndABadLanguage(t *testing.T) {
	foreign := serve(newTestRouter(&reportServiceStub{err: domain.ErrForbidden}),
		authorizedRequest(http.MethodGet, "/api/v1/reports/"+exportReportID+"/pdf", ""))
	if foreign.Code != http.StatusForbidden {
		t.Fatalf("foreign report status = %d", foreign.Code)
	}

	badLanguage := serve(newTestRouter(&reportServiceStub{report: exportReport()}),
		authorizedRequest(http.MethodGet, "/api/v1/reports/"+exportReportID+"/pdf?lang=de", ""))
	if badLanguage.Code != http.StatusBadRequest || !strings.Contains(badLanguage.Body.String(), "invalid_language") {
		t.Fatalf("bad language status = %d body = %s", badLanguage.Code, badLanguage.Body.String())
	}
}

func TestReportGeoJSONComesFromTheSnapshot(t *testing.T) {
	response := serve(newTestRouter(&reportServiceStub{report: exportReport()}),
		authorizedRequest(http.MethodGet, "/api/v1/reports/"+exportReportID+"/geojson", ""))

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || response.Code != http.StatusOK ||
		response.Header().Get("Content-Type") != "application/geo+json" || body["type"] != "FeatureCollection" {
		t.Fatalf("status = %d err = %v body = %s", response.Code, err, response.Body.String())
	}

	older := exportReport()
	older.Boundary = nil

	missing := serve(newTestRouter(&reportServiceStub{report: older}),
		authorizedRequest(http.MethodGet, "/api/v1/reports/"+exportReportID+"/geojson", ""))
	if missing.Code != http.StatusConflict || !strings.Contains(missing.Body.String(), "boundary_not_recorded") {
		t.Fatalf("older report status = %d body = %s", missing.Code, missing.Body.String())
	}
}

func TestReportShowsWhatItWasGeneratedFromAndTheLatestReport(t *testing.T) {
	report := exportReport()
	report.Climate = &domain.Climate{Status: domain.DataAvailable, FirstYear: 2016, LastYear: 2025}
	report.Terrain = &domain.Terrain{Status: domain.DataAvailable}
	report.Latest = &domain.LatestReport{ID: "newer-id", CreatedAt: report.CreatedAt.Add(time.Hour)}

	response := serve(newTestRouter(&reportServiceStub{report: report}),
		authorizedRequest(http.MethodGet, "/api/v1/reports/"+exportReportID, ""))
	body := response.Body.String()

	for _, fragment := range []string{
		`"generated_from":{"climate_first_year":2016,"climate_last_year":2025,"terrain_at":"2026-11-18T12:00:00Z"}`,
		`"latest_report_id":"newer-id"`,
		`"boundary":[{"lat":50.45,"lon":30.52}`,
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("body lacks %s: %s", fragment, body)
		}
	}
}
