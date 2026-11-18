package service

import (
	"context"
	"errors"
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const testListPlotID = "7d1f0a2e-3b4c-4d5e-8f60-718293a4b5c6"

func TestListReportsBuildsTheFilter(t *testing.T) {
	tests := []struct {
		name  string
		query ReportQuery
		want  domain.ReportFilter
	}{
		{"defaults", ReportQuery{}, domain.ReportFilter{OwnerID: "owner-1", Limit: domain.DefaultReportLimit}},
		{
			"plot and limit",
			ReportQuery{PlotID: testListPlotID, Limit: "5"},
			domain.ReportFilter{OwnerID: "owner-1", PlotID: testListPlotID, Limit: 5},
		},
		{"maximum limit", ReportQuery{Limit: "100"}, domain.ReportFilter{OwnerID: "owner-1", Limit: 100}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &reportRepositoryStub{}

			if _, err := NewReportService(repository, &plotClientStub{}, testInstanceID).
				List(context.Background(), "owner-1", test.query); err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if repository.filter != test.want {
				t.Fatalf("filter = %#v, want %#v", repository.filter, test.want)
			}
		})
	}
}

func TestListReportsRejectsInvalidQueryBeforeRepository(t *testing.T) {
	for _, query := range []ReportQuery{
		{Limit: "0"}, {Limit: "101"}, {Limit: "-1"}, {Limit: "ten"}, {Limit: "1.5"}, {PlotID: "plot-1"},
	} {
		repository := &reportRepositoryStub{}

		_, err := NewReportService(repository, &plotClientStub{}, testInstanceID).
			List(context.Background(), "owner-1", query)
		if !errors.Is(err, domain.ErrInvalidQuery) || repository.listed {
			t.Fatalf("query %#v: error = %v, repository called = %v", query, err, repository.listed)
		}
	}
}
