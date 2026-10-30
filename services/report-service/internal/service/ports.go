package service

import (
	"context"

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
	Apply(processingContext context.Context, event domain.PlotEvent) error
}
