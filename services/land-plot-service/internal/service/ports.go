package service

import (
	"context"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

type PlotRepository interface {
	Create(requestContext context.Context, plot domain.Plot) error
	GetByID(requestContext context.Context, id string) (domain.Plot, error)
	ListByOwner(requestContext context.Context, ownerID string) ([]domain.Plot, error)
	Update(requestContext context.Context, plot domain.Plot, expectedVersion int64) error
}

type EventPublisher interface {
	Publish(requestContext context.Context, event domain.PlotEvent) error
}
