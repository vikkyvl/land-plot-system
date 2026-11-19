package handler

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

type PlotService interface {
	Create(requestContext context.Context, draft domain.PlotDraft) (domain.Plot, error)
	List(requestContext context.Context, ownerID string) ([]domain.Plot, error)
	Summary(requestContext context.Context, ownerID string) (domain.PlotSummary, error)
	Get(requestContext context.Context, ownerID, plotID string) (domain.Plot, error)
	Update(requestContext context.Context, ownerID, plotID string, changes domain.PlotChanges) (domain.Plot, error)
	Delete(requestContext context.Context, ownerID, plotID string) error
	Versions(requestContext context.Context, ownerID, plotID string) ([]domain.PlotVersion, error)
	Preview(points []domain.Coordinate, survey domain.Survey) (domain.PolygonInspection, error)
}
