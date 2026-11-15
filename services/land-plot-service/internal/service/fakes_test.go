package service

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const testOwnerID = "owner-1"

var (
	errOutboxInsert = errors.New("outbox insert failed")
	errSagaStore    = errors.New("saga store failed")
)

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

type storeState struct {
	plots   map[string]domain.Plot
	sagas   map[string]domain.Saga
	reasons map[string]domain.CompensationReason
	events  []domain.PlotEvent
}

func (state storeState) clone() storeState {
	return storeState{
		plots:   maps.Clone(state.plots),
		sagas:   maps.Clone(state.sagas),
		reasons: maps.Clone(state.reasons),
		events:  slices.Clone(state.events),
	}
}

type fakeRepository struct {
	committed       storeState
	expiredSagaIDs  []string
	sagaTimeout     time.Duration
	failAppendEvent bool
	failSagaID      string
	err             error
}

func newFakeRepository(plots ...domain.Plot) *fakeRepository {
	repository := &fakeRepository{committed: storeState{
		plots:   map[string]domain.Plot{},
		sagas:   map[string]domain.Saga{},
		reasons: map[string]domain.CompensationReason{},
	}}
	for _, plot := range plots {
		repository.committed.plots[plot.ID] = plot
	}

	return repository
}

func (repository *fakeRepository) GetByID(_ context.Context, id string) (domain.Plot, error) {
	if repository.err != nil {
		return domain.Plot{}, repository.err
	}

	plot, ok := repository.committed.plots[id]
	if !ok {
		return domain.Plot{}, domain.ErrPlotNotFound
	}

	return plot, nil
}

func (repository *fakeRepository) ListByOwner(context.Context, string) ([]domain.Plot, error) {
	return slices.Collect(maps.Values(repository.committed.plots)), repository.err
}

func (repository *fakeRepository) ExpiredSagaIDs(context.Context, int) ([]string, error) {
	return repository.expiredSagaIDs, repository.err
}

func (repository *fakeRepository) InTransaction(_ context.Context, work func(store PlotStore) error) error {
	store := &fakeStore{repository: repository, state: repository.committed.clone()}
	if err := work(store); err != nil {
		return err
	}

	repository.committed = store.state

	return nil
}

type fakeStore struct {
	repository *fakeRepository
	state      storeState
}

func (store *fakeStore) CreatePlot(_ context.Context, plot domain.Plot) error {
	store.state.plots[plot.ID] = plot
	return nil
}

func (store *fakeStore) UpdatePlot(_ context.Context, plot domain.Plot, expectedVersion int64) error {
	if store.state.plots[plot.ID].Version != expectedVersion {
		return domain.ErrVersionConflict
	}

	store.state.plots[plot.ID] = plot

	return nil
}

func (store *fakeStore) LockPlot(_ context.Context, id string) (domain.Plot, error) {
	plot, ok := store.state.plots[id]
	if !ok {
		return domain.Plot{}, domain.ErrPlotNotFound
	}

	return plot, nil
}

func (store *fakeStore) StartSaga(_ context.Context, saga domain.Saga, timeout time.Duration) error {
	store.state.sagas[saga.ID] = saga
	store.repository.sagaTimeout = timeout

	return nil
}

func (store *fakeStore) FinishSaga(_ context.Context, finish SagaFinish) (domain.Saga, bool, error) {
	if finish.SagaID == store.repository.failSagaID {
		return domain.Saga{}, false, errSagaStore
	}

	saga, ok := store.state.sagas[finish.SagaID]
	if !ok || saga.Status != domain.SagaStarted {
		return domain.Saga{}, false, nil
	}

	saga.Status = finish.Status
	store.state.sagas[saga.ID] = saga
	store.state.reasons[saga.ID] = finish.Reason

	return saga, true, nil
}

func (store *fakeStore) AppendEvent(_ context.Context, event domain.PlotEvent) error {
	if store.repository.failAppendEvent {
		return errOutboxInsert
	}

	store.state.events = append(store.state.events, event)

	return nil
}

func testPoints() []domain.Coordinate {
	return []domain.Coordinate{
		{Latitude: 50, Longitude: 30},
		{Latitude: 50, Longitude: 30.001},
		{Latitude: 50.001, Longitude: 30},
	}
}

func testPlotDraft() domain.PlotDraft {
	return domain.PlotDraft{OwnerID: testOwnerID, Name: "Plot", Points: testPoints()}
}

func storedPlot(id string, status domain.PlotStatus) domain.Plot {
	return domain.Plot{
		ID:         id,
		OwnerID:    testOwnerID,
		Name:       "Plot",
		Points:     testPoints(),
		AreaM2:     100,
		PerimeterM: 40,
		Version:    3,
		Status:     status,
	}
}
