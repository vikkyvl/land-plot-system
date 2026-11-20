package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type PlotProjection struct {
	repository PlotSummaryRepository
	now        func() time.Time
}

func NewPlotProjection(repository PlotSummaryRepository) *PlotProjection {
	return &PlotProjection{repository: repository, now: time.Now}
}

func (projection *PlotProjection) Handle(processingContext context.Context, event domain.PlotEvent) error {
	reply := domain.NewSummaryApplied(uuid.NewString(), event, projection.now())
	if err := projection.repository.Apply(processingContext, event, reply); err != nil {
		return fmt.Errorf(wrapApplyPlotEvent, err)
	}

	return nil
}
