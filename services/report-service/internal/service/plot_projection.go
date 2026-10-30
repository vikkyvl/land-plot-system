package service

import (
	"context"
	"fmt"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type PlotProjection struct {
	repository PlotSummaryRepository
}

func NewPlotProjection(repository PlotSummaryRepository) *PlotProjection {
	return &PlotProjection{repository: repository}
}

func (projection *PlotProjection) Handle(processingContext context.Context, event domain.PlotEvent) error {
	if err := projection.repository.Apply(processingContext, event); err != nil {
		return fmt.Errorf(wrapApplyPlotEvent, err)
	}

	return nil
}
