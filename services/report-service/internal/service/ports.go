package service

import (
	"context"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type Caller struct {
	UserID      string
	AccessToken string
	RequestID   string
}

type ReportRepository interface {
	Create(requestContext context.Context, report domain.Report) error
	GetByID(requestContext context.Context, id string) (domain.Report, error)
}

type PlotClient interface {
	Get(requestContext context.Context, caller Caller, plotID string) (domain.PlotSnapshot, error)
	Trace(requestContext context.Context, requestID, message string) ([]domain.TraceHop, error)
}

type PlotSummaryRepository interface {
	Apply(processingContext context.Context, event domain.PlotEvent, reply domain.SummaryApplied) error
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
