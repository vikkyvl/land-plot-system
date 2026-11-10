package service

import (
	"context"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

type PlotRepository interface {
	GetByID(requestContext context.Context, id string) (domain.Plot, error)
	ListByOwner(requestContext context.Context, ownerID string) ([]domain.Plot, error)
	ExpiredSagaIDs(requestContext context.Context, limit int) ([]string, error)
	InTransaction(requestContext context.Context, work func(store PlotStore) error) error
}

type PlotStore interface {
	CreatePlot(requestContext context.Context, plot domain.Plot) error
	UpdatePlot(requestContext context.Context, plot domain.Plot, expectedVersion int64) error
	LockPlot(requestContext context.Context, id string) (domain.Plot, error)
	StartSaga(requestContext context.Context, saga domain.Saga, timeout time.Duration) error
	FinishSaga(requestContext context.Context, finish SagaFinish) (domain.Saga, bool, error)
	AppendEvent(requestContext context.Context, event domain.PlotEvent) error
}

type SagaFinish struct {
	SagaID string
	Status domain.SagaStatus
	Reason domain.CompensationReason
}

type OutboxRepository interface {
	PublishPending(requestContext context.Context, batch OutboxBatch, publish PublishFunc) (OutboxResult, error)
	DeletePublishedBefore(requestContext context.Context, age time.Duration) (int64, error)
}

type OutboxBatch struct {
	Limit       int
	MaxAttempts int
}

type OutboxResult struct {
	Published  int
	PublishErr error
	Abandoned  *AbandonedMessage
}

type AbandonedMessage struct {
	ID         string
	RoutingKey string
	Attempts   int
	LastError  string
}

type PublishFunc func(publishContext context.Context, message domain.OutboxMessage) error

type MessagePublisher interface {
	Publish(publishContext context.Context, message domain.OutboxMessage) error
}
