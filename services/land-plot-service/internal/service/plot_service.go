package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

type PlotService struct {
	repository  PlotRepository
	calculator  domain.AreaCalculator
	accuracy    domain.AccuracyModel
	sagaTimeout time.Duration
	now         func() time.Time
}

func NewPlotService(
	repository PlotRepository,
	calculator domain.AreaCalculator,
	accuracy domain.AccuracyModel,
	sagaTimeout time.Duration,
) *PlotService {
	return &PlotService{
		repository:  repository,
		calculator:  calculator,
		accuracy:    accuracy,
		sagaTimeout: sagaTimeout,
		now:         time.Now,
	}
}

func (plotService *PlotService) Create(requestContext context.Context, draft domain.PlotDraft) (domain.Plot, error) {
	plot, err := domain.NewPlot(uuid.NewString(), draft, plotService.calculator, plotService.now())
	if err != nil {
		return domain.Plot{}, err
	}

	event := domain.NewPlotEvent(uuid.NewString(), domain.PlotCreated, plot)
	saga := domain.NewCreateSaga(event.ID, plot)

	err = plotService.repository.InTransaction(requestContext, func(store PlotStore) error {
		if err := store.CreatePlot(requestContext, plot); err != nil {
			return err
		}

		return plotService.startSaga(requestContext, store, saga, event)
	})
	if err != nil {
		return domain.Plot{}, fmt.Errorf(wrapCreatePlot, err)
	}

	return plotService.withAccuracy(plot), nil
}

func (plotService *PlotService) Preview(
	points []domain.Coordinate,
	survey domain.Survey,
) (domain.PolygonInspection, error) {
	if err := survey.Validate(len(points)); err != nil {
		return domain.PolygonInspection{}, err
	}

	inspection := domain.InspectPolygon(points, plotService.calculator)
	if inspection.Valid() {
		accuracy := plotService.accuracy.Estimate(points, inspection.Area.AreaM2, survey)
		inspection.Accuracy = &accuracy
	}

	return inspection, nil
}

func (plotService *PlotService) List(requestContext context.Context, ownerID string) ([]domain.Plot, error) {
	plots, err := plotService.repository.ListByOwner(requestContext, ownerID)
	if err != nil {
		return nil, fmt.Errorf(wrapListPlots, err)
	}

	for i, plot := range plots {
		plots[i] = plotService.withAccuracy(plot)
	}

	return plots, nil
}

func (plotService *PlotService) Summary(requestContext context.Context, ownerID string) (domain.PlotSummary, error) {
	plots, err := plotService.repository.ListByOwner(requestContext, ownerID)
	if err != nil {
		return domain.PlotSummary{}, fmt.Errorf(wrapListPlots, err)
	}

	return domain.SummarizePlots(plots), nil
}

func (plotService *PlotService) Get(requestContext context.Context, ownerID, plotID string) (domain.Plot, error) {
	if _, err := uuid.Parse(plotID); err != nil {
		return domain.Plot{}, domain.ErrInvalidPlotID
	}

	plot, err := plotService.repository.GetByID(requestContext, plotID)
	if err != nil {
		return domain.Plot{}, fmt.Errorf(wrapGetPlot, err)
	}

	if plot.Status == domain.StatusCancelled {
		return domain.Plot{}, domain.ErrPlotNotFound
	}

	if plot.OwnerID != ownerID {
		return domain.Plot{}, domain.ErrForbidden
	}

	if plot.IsDeleted() {
		return domain.Plot{}, domain.ErrPlotDeleted
	}

	return plotService.withAccuracy(plot), nil
}

func (plotService *PlotService) Versions(
	requestContext context.Context,
	ownerID, plotID string,
) ([]domain.PlotVersion, error) {
	if _, err := plotService.Get(requestContext, ownerID, plotID); err != nil {
		return nil, err
	}

	versions, err := plotService.repository.ListVersions(requestContext, plotID)
	if err != nil {
		return nil, fmt.Errorf(wrapListPlotVersions, err)
	}

	for i, version := range versions {
		versions[i].Accuracy = plotService.accuracy.Estimate(version.Points, version.AreaM2, version.Survey)
	}

	return versions, nil
}

func (plotService *PlotService) Delete(requestContext context.Context, ownerID, plotID string) error {
	plot, err := plotService.Get(requestContext, ownerID, plotID)
	if err != nil {
		return err
	}

	deleted, err := plot.Deleted(plotService.now())
	if err != nil {
		return err
	}

	event := domain.NewPlotEvent(uuid.NewString(), domain.PlotDeleted, deleted)

	err = plotService.repository.InTransaction(requestContext, func(store PlotStore) error {
		if err := store.UpdatePlot(requestContext, deleted, plot.Version); err != nil {
			return err
		}

		return store.AppendEvent(requestContext, event)
	})
	if err != nil {
		return fmt.Errorf(wrapDeletePlot, err)
	}

	return nil
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

	event := domain.NewPlotEvent(uuid.NewString(), domain.PlotUpdated, updated)
	saga := domain.NewUpdateSaga(event.ID, plot, updated)

	err = plotService.repository.InTransaction(requestContext, func(store PlotStore) error {
		if err := store.UpdatePlot(requestContext, updated, plot.Version); err != nil {
			return err
		}

		return plotService.startSaga(requestContext, store, saga, event)
	})
	if err != nil {
		return domain.Plot{}, fmt.Errorf(wrapUpdatePlot, err)
	}

	return plotService.withAccuracy(updated), nil
}

func (plotService *PlotService) withAccuracy(plot domain.Plot) domain.Plot {
	plot.Accuracy = plotService.accuracy.Estimate(plot.Points, plot.AreaM2, plot.Survey)

	return plot
}

func (plotService *PlotService) startSaga(
	requestContext context.Context,
	store PlotStore,
	saga domain.Saga,
	event domain.PlotEvent,
) error {
	if err := store.StartSaga(requestContext, saga, plotService.sagaTimeout); err != nil {
		return err
	}

	return store.AppendEvent(requestContext, event)
}
