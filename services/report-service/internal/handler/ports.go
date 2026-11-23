package handler

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

type ExportService interface {
	PDF(requestContext context.Context, ownerID, reportID, language string) (domain.ReportFile, error)
	GeoJSON(requestContext context.Context, ownerID, reportID, language string) (domain.ReportFile, error)
}

type ReportService interface {
	Create(requestContext context.Context, caller service.Caller, plotID string) (domain.Report, error)
	Get(requestContext context.Context, ownerID, reportID string) (domain.Report, error)
	List(requestContext context.Context, ownerID string, query service.ReportQuery) ([]domain.Report, error)
	Trace(requestContext context.Context, caller service.Caller, gatewayInstance, message string) (domain.Trace, error)
}
