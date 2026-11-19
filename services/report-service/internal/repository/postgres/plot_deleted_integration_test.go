//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func applyPlotEvent(t *testing.T, repository *PlotSummaryRepository, event domain.PlotEvent) {
	t.Helper()

	reply := domain.NewSummaryApplied(uuid.NewString(), event, time.Now())
	if err := repository.Apply(context.Background(), event, reply); err != nil {
		t.Fatalf("Apply(%s) error = %v", event.Type, err)
	}
}

func storeReport(t *testing.T, pool *pgxpool.Pool, plotID, ownerID string) string {
	t.Helper()

	report := domain.NewReport(uuid.NewString(), domain.PlotSnapshot{
		ID: plotID, OwnerID: ownerID, Name: "Plot", AreaM2: 100, PerimeterM: 40, Version: 1,
	}, time.Now())
	if err := NewReportRepository(pool).Create(context.Background(), report); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	return report.ID
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, arguments ...any) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(context.Background(), query, arguments...).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}

	return count
}

func TestPlotDeletedMarksReportsAndKeepsThem(t *testing.T) {
	pool := openTestPool(t)
	summaries := NewPlotSummaryRepository(pool, "")
	plotID, ownerID, otherPlotID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	occurredAt := time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)

	created := domain.PlotEvent{
		ID: uuid.NewString(), Type: domain.PlotCreated, OccurredAt: occurredAt,
		PlotID: plotID, OwnerID: ownerID, Name: "Plot", AreaM2: 100, Version: 1,
	}
	applyPlotEvent(t, summaries, created)

	reportID := storeReport(t, pool, plotID, ownerID)
	otherReportID := storeReport(t, pool, otherPlotID, ownerID)

	deleted := created
	deleted.ID, deleted.Type, deleted.Version = uuid.NewString(), domain.PlotDeleted, 2
	deleted.OccurredAt = occurredAt.Add(time.Hour)
	applyPlotEvent(t, summaries, deleted)
	applyPlotEvent(t, summaries, deleted)

	reports := NewReportRepository(pool)

	marked, err := reports.GetByID(context.Background(), reportID)
	if err != nil || !marked.PlotDeleted() || !marked.PlotDeletedAt.Equal(deleted.OccurredAt) {
		t.Fatalf("report of the deleted plot = %#v, %v", marked, err)
	}

	untouched, err := reports.GetByID(context.Background(), otherReportID)
	if err != nil || untouched.PlotDeleted() {
		t.Fatalf("report of another plot = %#v, %v", untouched, err)
	}

	if countRows(t, pool, `SELECT count(*) FROM plot_summaries WHERE plot_id = $1 AND deleted_at IS NOT NULL`,
		plotID) != 1 {
		t.Fatal("plot summary is not marked as deleted")
	}

	if replies := countRows(t, pool, `SELECT count(*) FROM outbox_events`); replies != 2 {
		t.Fatalf("summary replies = %d, want one per distinct event", replies)
	}
}
