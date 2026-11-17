//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

func storeConfirmedPlot(t *testing.T, repository *PlotRepository, ownerID string) domain.Plot {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Microsecond)
	plot := domain.Plot{
		ID:         uuid.NewString(),
		OwnerID:    ownerID,
		Name:       "Plot",
		Points:     []domain.Coordinate{{Latitude: 50, Longitude: 30}},
		AreaM2:     100,
		PerimeterM: 40,
		Version:    1,
		Status:     domain.StatusConfirmed,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := repository.InTransaction(context.Background(), func(store service.PlotStore) error {
		return store.CreatePlot(context.Background(), plot)
	})
	if err != nil {
		t.Fatalf("store plot: %v", err)
	}

	return plot
}

func TestDeletedPlotKeepsItsRowAndLeavesTheList(t *testing.T) {
	repository := NewPlotRepository(openTestPool(t))
	ownerID := uuid.NewString()
	kept := storeConfirmedPlot(t, repository, ownerID)
	removed := storeConfirmedPlot(t, repository, ownerID)

	deleted, err := removed.Deleted(time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatalf("Deleted() error = %v", err)
	}

	err = repository.InTransaction(context.Background(), func(store service.PlotStore) error {
		return store.UpdatePlot(context.Background(), deleted, removed.Version)
	})
	if err != nil {
		t.Fatalf("UpdatePlot() error = %v", err)
	}

	stored, err := repository.GetByID(context.Background(), removed.ID)
	if err != nil || stored.DeletedAt == nil || !stored.DeletedAt.Equal(*deleted.DeletedAt) || stored.Version != 2 {
		t.Fatalf("GetByID() = %#v, %v", stored, err)
	}

	listed, err := repository.ListByOwner(context.Background(), ownerID)
	if err != nil || len(listed) != 1 || listed[0].ID != kept.ID {
		t.Fatalf("ListByOwner() = %#v, %v", listed, err)
	}

	err = repository.InTransaction(context.Background(), func(store service.PlotStore) error {
		return store.UpdatePlot(context.Background(), deleted, deleted.Version)
	})
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("UpdatePlot() of a deleted plot error = %v, want ErrVersionConflict", err)
	}
}
