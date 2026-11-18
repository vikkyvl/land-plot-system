//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

func TestRecordedVersionsAreListedNewestFirstOnce(t *testing.T) {
	repository := NewPlotRepository(openTestPool(t))
	plot := storeConfirmedPlot(t, repository, uuid.NewString())
	second := plot
	second.Version, second.Name = 2, "Renamed"
	second.UpdatedAt = plot.UpdatedAt.Add(time.Hour)

	err := repository.InTransaction(context.Background(), func(store service.PlotStore) error {
		for _, version := range []domain.PlotVersion{
			plot.ConfirmedVersion(), second.ConfirmedVersion(), second.ConfirmedVersion(),
		} {
			if err := store.RecordVersion(context.Background(), version); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("RecordVersion() error = %v", err)
	}

	versions, err := repository.ListVersions(context.Background(), plot.ID)
	if err != nil || len(versions) != 2 || versions[0].Version != 2 || versions[0].Name != "Renamed" ||
		!versions[0].CreatedAt.Equal(second.UpdatedAt) || versions[1].Version != 1 || len(versions[1].Points) != 1 {
		t.Fatalf("ListVersions() = %#v, %v", versions, err)
	}
}

func TestMigrationBackfillsCurrentVersionOfConfirmedPlots(t *testing.T) {
	pool := openTestPool(t)
	repository := NewPlotRepository(pool)
	confirmed := storeConfirmedPlot(t, repository, uuid.NewString())
	deleted := storeConfirmedPlot(t, repository, uuid.NewString())

	if _, err := pool.Exec(
		context.Background(),
		`UPDATE plots SET deleted_at = now() WHERE id = $1`,
		deleted.ID,
	); err != nil {
		t.Fatalf("mark deleted: %v", err)
	}

	if err := repository.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	backfilled, err := repository.ListVersions(context.Background(), confirmed.ID)
	if err != nil || len(backfilled) != 1 || backfilled[0].Version != confirmed.Version {
		t.Fatalf("ListVersions(confirmed) = %#v, %v", backfilled, err)
	}

	if skipped, err := repository.ListVersions(context.Background(), deleted.ID); err != nil || len(skipped) != 0 {
		t.Fatalf("ListVersions(deleted) = %#v, %v", skipped, err)
	}
}

func TestPlotsAreListedByLatestChangeFirst(t *testing.T) {
	repository := NewPlotRepository(openTestPool(t))
	ownerID := uuid.NewString()
	older := storeConfirmedPlot(t, repository, ownerID)
	newer := storeConfirmedPlot(t, repository, ownerID)

	renamed := older
	renamed.Version, renamed.UpdatedAt = 2, newer.UpdatedAt.Add(time.Hour)

	err := repository.InTransaction(context.Background(), func(store service.PlotStore) error {
		return store.UpdatePlot(context.Background(), renamed, older.Version)
	})
	if err != nil {
		t.Fatalf("UpdatePlot() error = %v", err)
	}

	plots, err := repository.ListByOwner(context.Background(), ownerID)
	if err != nil || len(plots) != 2 || plots[0].ID != older.ID || plots[1].ID != newer.ID {
		t.Fatalf("ListByOwner() = %#v, %v", plots, err)
	}
}
