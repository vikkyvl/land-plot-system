//go:build integration

package postgres

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func TestReportKeepsAccuracyTerrainAndClimateSnapshot(t *testing.T) {
	repository := NewReportRepository(openTestPool(t))
	worstSide := 1
	createdAt := time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
	report := domain.Report{
		ID: uuid.NewString(), OwnerID: uuid.NewString(), PlotID: uuid.NewString(), PlotName: "Plot", PlotVersion: 3,
		AreaM2: 24_800, AreaAres: 248, AreaHectares: 2.48, PerimeterM: 630, CreatedAt: createdAt,
		Accuracy: &domain.AreaAccuracy{
			AreaMarginM2: 520, AreaMarginHectares: 0.052, Percent: 2.1, Confidence: 0.95,
			ScaleShare: 0.4, ObstructionShare: 0.6, WorstSide: &worstSide, GroundShiftM: 4, Basis: "capture",
		},
	}.WithSurroundings(
		domain.Terrain{
			Status:        domain.DataAvailable,
			SlopeDegrees:  16.7,
			SurfaceAreaM2: 25_900,
			ReliefM:       231,
			Coarse:        true,
		},
		domain.UnavailableClimate(domain.ClimatePeriod{FirstYear: 2016, LastYear: 2025}),
	)
	older := domain.Report{
		ID: uuid.NewString(), OwnerID: report.OwnerID, PlotID: report.PlotID, PlotName: "Plot", PlotVersion: 1,
		AreaM2: 100, AreaAres: 1, AreaHectares: 0.01, PerimeterM: 40, CreatedAt: createdAt.Add(-time.Hour),
	}

	for _, stored := range []domain.Report{report, older} {
		if err := repository.Create(context.Background(), stored); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	loaded, err := repository.GetByID(context.Background(), report.ID)
	loaded.CreatedAt = loaded.CreatedAt.UTC()

	if err != nil || !reflect.DeepEqual(loaded, report) {
		t.Fatalf("GetByID() = %+v, %v\nwant %+v", loaded, err, report)
	}

	loadedOlder, err := repository.GetByID(context.Background(), older.ID)
	if err != nil || loadedOlder.Accuracy != nil || loadedOlder.Terrain != nil || loadedOlder.Climate != nil {
		t.Fatalf("older report = %+v, %v", loadedOlder, err)
	}
}
