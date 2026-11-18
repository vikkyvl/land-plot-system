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
	List(requestContext context.Context, filter domain.ReportFilter) ([]domain.Report, error)
}

type PlotClient interface {
	Get(requestContext context.Context, caller Caller, plotID string) (domain.PlotSnapshot, error)
	Trace(requestContext context.Context, caller Caller, message string) ([]domain.TraceHop, error)
}

type ElevationProvider interface {
	Elevations(requestContext context.Context, points []domain.Coordinate) ([]float64, error)
}

type ClimateProvider interface {
	DailyWeather(
		requestContext context.Context,
		point domain.Coordinate,
		from, to time.Time,
	) ([]domain.DailyWeather, error)
}

type ClimateCache interface {
	Get(requestContext context.Context, key ClimateCacheKey, notBefore time.Time) (domain.Climate, bool, error)
	Put(requestContext context.Context, key ClimateCacheKey, climate domain.Climate, fetchedAt time.Time) error
}

type ClimateCacheKey struct {
	Cell   domain.ClimateCell
	Period domain.ClimatePeriod
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
