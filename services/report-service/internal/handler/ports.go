package handler

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

type ReportService interface {
	Create(requestContext context.Context, caller service.Caller, plotID string) (domain.Report, error)
	Get(requestContext context.Context, ownerID, reportID string) (domain.Report, error)
	Trace(requestContext context.Context, gatewayInstance, requestID, message string) (domain.Trace, error)
}
