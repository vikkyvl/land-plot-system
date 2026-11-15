package domain

import (
	"testing"
	"time"
)

func TestNewReportConvertsAreaUnits(t *testing.T) {
	now := time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC)
	plot := PlotSnapshot{
		ID: "plot-1", OwnerID: "owner-1", Name: "Plot", AreaM2: 12_500,
		PerimeterM: 450, Version: 3,
	}
	report := NewReport("report-1", plot, now)

	if report.AreaM2 != 12_500 || report.AreaAres != 125 || report.AreaHectares != 1.25 {
		t.Fatalf("area conversions = m2:%f ares:%f hectares:%f", report.AreaM2, report.AreaAres, report.AreaHectares)
	}

	if report.PlotVersion != 3 || report.PerimeterM != 450 || !report.CreatedAt.Equal(now) {
		t.Fatalf("report = %#v", report)
	}
}
