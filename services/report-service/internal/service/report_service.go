package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	gatewayServiceName = "api-gateway"
	reportServiceName  = "report-service"
	traceProtocol      = "HTTP"
)

type ReportService struct {
	repository   ReportRepository
	plots        PlotClient
	surroundings Surroundings
	instanceID   string
	now          func() time.Time
}

func NewReportService(
	repository ReportRepository,
	plots PlotClient,
	surroundings Surroundings,
	instanceID string,
) *ReportService {
	return &ReportService{
		repository:   repository,
		plots:        plots,
		surroundings: surroundings,
		instanceID:   instanceID,
		now:          time.Now,
	}
}

func (reportService *ReportService) Create(
	requestContext context.Context,
	caller Caller,
	plotID string,
) (domain.Report, error) {
	if _, err := uuid.Parse(plotID); err != nil {
		return domain.Report{}, domain.ErrInvalidPlotID
	}

	plot, err := reportService.plots.Get(requestContext, caller, plotID)
	if err != nil {
		return domain.Report{}, fmt.Errorf(wrapGetPlotForReport, err)
	}

	if plot.OwnerID != caller.UserID {
		return domain.Report{}, domain.ErrForbidden
	}

	if !plot.IsConfirmed() {
		return domain.Report{}, domain.ErrPlotNotConfirmed
	}

	terrain, climate := reportService.surroundings.Describe(requestContext, plot)
	report := domain.NewReport(uuid.NewString(), plot, reportService.now()).WithSurroundings(terrain, climate)

	if err := reportService.repository.Create(requestContext, report); err != nil {
		return domain.Report{}, fmt.Errorf(wrapCreateReport, err)
	}

	return report, nil
}

func (reportService *ReportService) Get(
	requestContext context.Context,
	ownerID, reportID string,
) (domain.Report, error) {
	if _, err := uuid.Parse(reportID); err != nil {
		return domain.Report{}, domain.ErrInvalidReportID
	}

	report, err := reportService.repository.GetByID(requestContext, reportID)
	if err != nil {
		return domain.Report{}, fmt.Errorf(wrapGetReport, err)
	}

	if report.OwnerID != ownerID {
		return domain.Report{}, domain.ErrForbidden
	}

	return report, nil
}

func (reportService *ReportService) List(
	requestContext context.Context,
	ownerID string,
	query ReportQuery,
) ([]domain.Report, error) {
	filter, err := newReportFilter(ownerID, query)
	if err != nil {
		return nil, err
	}

	reports, err := reportService.repository.List(requestContext, filter)
	if err != nil {
		return nil, fmt.Errorf(wrapListReports, err)
	}

	return reports, nil
}

func (reportService *ReportService) Trace(
	requestContext context.Context,
	caller Caller,
	gatewayInstance, message string,
) (domain.Trace, error) {
	landPath, err := reportService.plots.Trace(requestContext, caller, message)
	if err != nil {
		return domain.Trace{}, fmt.Errorf(wrapTraceLandPlotService, err)
	}

	path := []domain.TraceHop{
		{Service: gatewayServiceName, Instance: gatewayInstance, Protocol: traceProtocol},
		{Service: reportServiceName, Instance: reportService.instanceID, Protocol: traceProtocol},
	}

	return domain.Trace{Message: message, Path: append(path, landPath...)}, nil
}
