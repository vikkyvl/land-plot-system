package handler

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

type PlotService interface {
	Create(requestContext context.Context, draft domain.PlotDraft) (domain.Plot, error)
	List(requestContext context.Context, ownerID string) ([]domain.Plot, error)
	Get(requestContext context.Context, ownerID, plotID string) (domain.Plot, error)
	Update(requestContext context.Context, ownerID, plotID string, changes domain.PlotChanges) (domain.Plot, error)
}
