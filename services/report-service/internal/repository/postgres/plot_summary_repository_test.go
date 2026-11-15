package postgres

import (
	"testing"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func TestPlotSummaryRepositoryFailsOnlyPlotChangesWithPrefix(t *testing.T) {
	repository := &PlotSummaryRepository{failNamePrefix: "FAIL"}
	tests := []struct {
		event domain.PlotEvent
		want  bool
	}{
		{domain.PlotEvent{Type: domain.PlotCreated, Name: "FAIL field"}, true},
		{domain.PlotEvent{Type: domain.PlotUpdated, Name: "FAIL renamed"}, true},
		{domain.PlotEvent{Type: domain.PlotCancelled, Name: "FAIL field"}, false},
		{domain.PlotEvent{Type: domain.PlotCreated, Name: "North field"}, false},
	}

	for _, test := range tests {
		if got := repository.mustFail(test.event); got != test.want {
			t.Fatalf("mustFail(%s %q) = %v, want %v", test.event.Type, test.event.Name, got, test.want)
		}
	}
}

func TestPlotSummaryRepositoryWithoutPrefixNeverFails(t *testing.T) {
	repository := &PlotSummaryRepository{}

	if repository.mustFail(domain.PlotEvent{Type: domain.PlotCreated, Name: "FAIL field"}) {
		t.Fatal("mustFail() = true without a configured prefix")
	}
}
