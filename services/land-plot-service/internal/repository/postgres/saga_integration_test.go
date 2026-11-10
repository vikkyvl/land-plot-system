//go:build integration

package postgres

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

func startTestSaga(t *testing.T, repository *PlotRepository, timeout time.Duration) string {
	t.Helper()

	now := time.Now().UTC()
	plot := domain.Plot{
		ID:         uuid.NewString(),
		OwnerID:    uuid.NewString(),
		Name:       "Plot",
		Points:     []domain.Coordinate{{Latitude: 50, Longitude: 30}},
		AreaM2:     100,
		PerimeterM: 40,
		Version:    1,
		Status:     domain.StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	saga := domain.NewCreateSaga(uuid.NewString(), plot)

	err := repository.InTransaction(context.Background(), func(store service.PlotStore) error {
		if err := store.CreatePlot(context.Background(), plot); err != nil {
			return err
		}

		return store.StartSaga(context.Background(), saga, timeout)
	})
	if err != nil {
		t.Fatalf("start saga: %v", err)
	}

	return saga.ID
}

func finishTestSaga(t *testing.T, repository *PlotRepository, finish service.SagaFinish) (domain.Saga, bool) {
	t.Helper()

	var (
		saga    domain.Saga
		started bool
	)

	err := repository.InTransaction(context.Background(), func(store service.PlotStore) error {
		var err error

		saga, started, err = store.FinishSaga(context.Background(), finish)

		return err
	})
	if err != nil {
		t.Fatalf("FinishSaga() error = %v", err)
	}

	return saga, started
}

func TestFinishSagaChangesOnlyStartedSaga(t *testing.T) {
	repository := NewPlotRepository(openTestPool(t))
	sagaID := startTestSaga(t, repository, time.Hour)

	saga, started := finishTestSaga(t, repository, service.SagaFinish{SagaID: sagaID, Status: domain.SagaCompleted})
	if !started || saga.ID != sagaID || saga.Status != domain.SagaCompleted || saga.Kind != domain.SagaCreate {
		t.Fatalf("first FinishSaga() = %#v, started = %v", saga, started)
	}

	_, started = finishTestSaga(t, repository, service.SagaFinish{
		SagaID: sagaID,
		Status: domain.SagaCompensated,
		Reason: domain.ReasonTimeout,
	})
	if started {
		t.Fatal("second FinishSaga() changed a finished saga")
	}
}

func TestExpiredSagaIDsReturnsOnlyStartedSagasPastDeadline(t *testing.T) {
	repository := NewPlotRepository(openTestPool(t))
	expired := startTestSaga(t, repository, -time.Second)
	startTestSaga(t, repository, time.Hour)

	finished := startTestSaga(t, repository, -time.Second)
	finishTestSaga(t, repository, service.SagaFinish{SagaID: finished, Status: domain.SagaCompleted})

	sagaIDs, err := repository.ExpiredSagaIDs(context.Background(), 10)
	if err != nil {
		t.Fatalf("ExpiredSagaIDs() error = %v", err)
	}

	if !slices.Equal(sagaIDs, []string{expired}) {
		t.Fatalf("ExpiredSagaIDs() = %v, want [%s]", sagaIDs, expired)
	}
}
