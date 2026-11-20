//go:build integration

package postgres

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

func TestSurveyIsStoredWithPlotVersionAndSagaState(t *testing.T) {
	repository := NewPlotRepository(openTestPool(t))
	now := time.Now().UTC().Truncate(time.Microsecond)
	survey := domain.Survey{
		Capture: &domain.Capture{Zoom: 17.5, Layer: domain.LayerSatellite, Pointer: domain.PointerTouch},
		Obstructions: []domain.Obstruction{
			{Side: 0, Reason: domain.ReasonTrees},
			{Side: 2, Reason: domain.ReasonUnclear},
		},
	}
	plot := domain.Plot{
		ID:      uuid.NewString(),
		OwnerID: uuid.NewString(),
		Name:    "Plot",
		Points: []domain.Coordinate{
			{Latitude: 50, Longitude: 30},
			{Latitude: 50, Longitude: 30.001},
			{Latitude: 50.001, Longitude: 30},
		},
		AreaM2:     100,
		PerimeterM: 40,
		Survey:     survey,
		Version:    1,
		Status:     domain.StatusConfirmed,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	updated := plot
	updated.Survey = domain.Survey{}
	updated.Version = 2
	saga := domain.NewUpdateSaga(uuid.NewString(), plot, updated)

	err := repository.InTransaction(context.Background(), func(store service.PlotStore) error {
		if err := store.CreatePlot(context.Background(), plot); err != nil {
			return err
		}

		if err := store.RecordVersion(context.Background(), plot.ConfirmedVersion()); err != nil {
			return err
		}

		if err := store.UpdatePlot(context.Background(), updated, plot.Version); err != nil {
			return err
		}

		return store.StartSaga(context.Background(), saga, time.Minute)
	})
	if err != nil {
		t.Fatalf("store plot: %v", err)
	}

	stored, err := repository.GetByID(context.Background(), plot.ID)
	if err != nil || stored.Survey.Capture != nil || stored.Survey.Obstructions != nil {
		t.Fatalf("GetByID() survey = %+v, %v", stored.Survey, err)
	}

	versions, err := repository.ListVersions(context.Background(), plot.ID)
	if err != nil || len(versions) != 1 || !reflect.DeepEqual(versions[0].Survey, survey) {
		t.Fatalf("ListVersions() = %+v, %v", versions, err)
	}

	finished, _ := finishTestSaga(t, repository, service.SagaFinish{SagaID: saga.ID, Status: domain.SagaCompensated})
	if finished.PreviousState == nil || !reflect.DeepEqual(finished.PreviousState.Survey, survey) {
		t.Fatalf("saga previous state = %+v", finished.PreviousState)
	}
}
