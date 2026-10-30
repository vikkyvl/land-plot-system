package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

type PlotService struct {
	repository PlotRepository
	publisher  EventPublisher
	calculator domain.AreaCalculator
	logger     *slog.Logger
	now        func() time.Time
}

func NewPlotService(
	repository PlotRepository,
	publisher EventPublisher,
	calculator domain.AreaCalculator,
	logger *slog.Logger,
) *PlotService {
	return &PlotService{
		repository: repository,
		publisher:  publisher,
		calculator: calculator,
		logger:     logger,
		now:        time.Now,
	}
}

func (plotService *PlotService) Create(requestContext context.Context, draft domain.PlotDraft) (domain.Plot, error) {
	plot, err := domain.NewPlot(uuid.NewString(), draft, plotService.calculator, plotService.now())
	if err != nil {
		return domain.Plot{}, err
	}

	if err := plotService.repository.Create(requestContext, plot); err != nil {
		return domain.Plot{}, fmt.Errorf(wrapCreatePlot, err)
	}

	plotService.publish(requestContext, domain.PlotCreated, plot)

	return plot, nil
}

func (plotService *PlotService) List(requestContext context.Context, ownerID string) ([]domain.Plot, error) {
	plots, err := plotService.repository.ListByOwner(requestContext, ownerID)
	if err != nil {
		return nil, fmt.Errorf(wrapListPlots, err)
	}

	return plots, nil
}

func (plotService *PlotService) Get(requestContext context.Context, ownerID, plotID string) (domain.Plot, error) {
	if _, err := uuid.Parse(plotID); err != nil {
		return domain.Plot{}, domain.ErrInvalidPlotID
	}

	plot, err := plotService.repository.GetByID(requestContext, plotID)
	if err != nil {
		return domain.Plot{}, fmt.Errorf(wrapGetPlot, err)
	}

	if plot.OwnerID != ownerID {
		return domain.Plot{}, domain.ErrForbidden
	}

	return plot, nil
}

func (plotService *PlotService) Update(
	requestContext context.Context,
	ownerID, plotID string,
	changes domain.PlotChanges,
) (domain.Plot, error) {
	plot, err := plotService.Get(requestContext, ownerID, plotID)
	if err != nil {
		return domain.Plot{}, err
	}

	updated, err := plot.Updated(changes, plotService.calculator, plotService.now())
	if err != nil {
		return domain.Plot{}, err
	}

	if err := plotService.repository.Update(requestContext, updated, plot.Version); err != nil {
		return domain.Plot{}, fmt.Errorf(wrapUpdatePlot, err)
	}

	plotService.publish(requestContext, domain.PlotUpdated, updated)

	return updated, nil
}

func (plotService *PlotService) publish(
	requestContext context.Context,
	eventType domain.PlotEventType,
	plot domain.Plot,
) {
	event := domain.NewPlotEvent(uuid.NewString(), eventType, plot)
	if err := plotService.publisher.Publish(requestContext, event); err != nil {
		plotService.logger.ErrorContext(
			requestContext,
			logPublishPlotEvent,
			logging.KeyPlotID, plot.ID,
			logging.KeyEventType, eventType,
			logging.KeyError, err,
		)
	}
}
