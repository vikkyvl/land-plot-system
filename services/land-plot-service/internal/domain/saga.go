package domain

import "time"

type SagaKind string

const (
	SagaCreate SagaKind = "CREATE"
	SagaUpdate SagaKind = "UPDATE"
)

type SagaStatus string

const (
	SagaStarted     SagaStatus = "STARTED"
	SagaCompleted   SagaStatus = "COMPLETED"
	SagaCompensated SagaStatus = "COMPENSATED"
)

type CompensationReason string

const (
	ReasonDeadLetter CompensationReason = "dlq"
	ReasonTimeout    CompensationReason = "timeout"
)

type Saga struct {
	ID            string
	PlotID        string
	Kind          SagaKind
	Status        SagaStatus
	PreviousState *PlotState
	NewVersion    int64
}

func NewCreateSaga(eventID string, created Plot) Saga {
	return Saga{
		ID:         eventID,
		PlotID:     created.ID,
		Kind:       SagaCreate,
		Status:     SagaStarted,
		NewVersion: created.Version,
	}
}

func NewUpdateSaga(eventID string, previous, updated Plot) Saga {
	previousState := previous.State()

	return Saga{
		ID:            eventID,
		PlotID:        updated.ID,
		Kind:          SagaUpdate,
		Status:        SagaStarted,
		PreviousState: &previousState,
		NewVersion:    updated.Version,
	}
}

func (saga Saga) Compensate(plot Plot, now time.Time) (Plot, PlotEventType) {
	if saga.Kind == SagaUpdate && saga.PreviousState != nil {
		return plot.Restored(*saga.PreviousState, now), PlotUpdated
	}

	return plot.Cancelled(now), PlotCancelled
}
