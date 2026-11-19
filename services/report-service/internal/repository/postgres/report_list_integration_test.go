//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func TestReportsAreListedNewestFirstForTheOwner(t *testing.T) {
	repository := NewReportRepository(openTestPool(t))
	ownerID, plotID, otherPlotID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	start := time.Date(2026, time.November, 18, 9, 0, 0, 0, time.UTC)

	var ids []string

	for index, report := range []struct {
		ownerID string
		plotID  string
	}{
		{ownerID, plotID}, {ownerID, otherPlotID}, {ownerID, plotID}, {uuid.NewString(), plotID},
	} {
		created := domain.NewReport(uuid.NewString(), domain.PlotSnapshot{
			ID: report.plotID, OwnerID: report.ownerID, Name: "Plot", AreaM2: 100, PerimeterM: 40, Version: 1,
		}, start.Add(time.Duration(index)*time.Minute))
		if err := repository.Create(context.Background(), created); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		ids = append(ids, created.ID)
	}

	tests := []struct {
		name   string
		filter domain.ReportFilter
		want   []string
	}{
		{"all of the owner", domain.ReportFilter{OwnerID: ownerID, Limit: 50}, []string{ids[2], ids[1], ids[0]}},
		{"one plot", domain.ReportFilter{OwnerID: ownerID, PlotID: plotID, Limit: 50}, []string{ids[2], ids[0]}},
		{"limited", domain.ReportFilter{OwnerID: ownerID, Limit: 1}, []string{ids[2]}},
		{"nothing", domain.ReportFilter{OwnerID: uuid.NewString(), Limit: 50}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reports, err := repository.List(context.Background(), test.filter)
			if err != nil || len(reports) != len(test.want) {
				t.Fatalf("List() = %#v, %v", reports, err)
			}

			for index, report := range reports {
				if report.ID != test.want[index] || report.PlotDeleted() {
					t.Fatalf("report %d = %#v, want ID %s", index, report, test.want[index])
				}
			}
		})
	}
}
