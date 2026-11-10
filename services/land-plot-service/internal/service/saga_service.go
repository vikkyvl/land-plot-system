package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

const expiredSagaBatchSize = 50

type SagaService struct {
	repository PlotRepository
	logger     *slog.Logger
	now        func() time.Time
}

func NewSagaService(repository PlotRepository, logger *slog.Logger) *SagaService {
	return &SagaService{repository: repository, logger: logger, now: time.Now}
}

func (sagaService *SagaService) Confirm(processingContext context.Context, sagaID string) error {
	err := sagaService.repository.InTransaction(processingContext, func(store PlotStore) error {
		saga, started, err := store.FinishSaga(processingContext, SagaFinish{
			SagaID: sagaID,
			Status: domain.SagaCompleted,
		})
		if err != nil || !started {
			return err
		}

		plot, err := store.LockPlot(processingContext, saga.PlotID)
		if err != nil {
			return err
		}

		if err := store.UpdatePlot(processingContext, plot.Confirmed(), plot.Version); err != nil {
			return err
		}

		sagaService.logger.Info(
			logSagaCompleted,
			logging.KeySagaID, saga.ID,
			logging.KeyPlotID, saga.PlotID,
		)

		return nil
	})
	if err != nil {
		return fmt.Errorf(wrapConfirmSaga, err)
	}

	return nil
}

func (sagaService *SagaService) Compensate(
	processingContext context.Context,
	sagaID string,
	reason domain.CompensationReason,
) error {
	_, err := sagaService.compensateInTransaction(processingContext, sagaID, reason)

	return err
}

func (sagaService *SagaService) CompensateExpired(processingContext context.Context) (int, error) {
	sagaIDs, err := sagaService.repository.ExpiredSagaIDs(processingContext, expiredSagaBatchSize)
	if err != nil {
		return 0, fmt.Errorf(wrapCompensateExpiredSagas, err)
	}

	var (
		compensated int
		failures    []error
	)

	for _, sagaID := range sagaIDs {
		started, err := sagaService.compensateInTransaction(processingContext, sagaID, domain.ReasonTimeout)
		if err != nil {
			sagaService.logger.Error(logCompensateExpiredSaga, logging.KeySagaID, sagaID, logging.KeyError, err)
			failures = append(failures, err)

			continue
		}

		if started {
			compensated++
		}
	}

	return compensated, errors.Join(failures...)
}

func (sagaService *SagaService) compensateInTransaction(
	processingContext context.Context,
	sagaID string,
	reason domain.CompensationReason,
) (bool, error) {
	var started bool

	err := sagaService.repository.InTransaction(processingContext, func(store PlotStore) error {
		var err error

		started, err = sagaService.compensate(processingContext, store, sagaID, reason)

		return err
	})
	if err != nil {
		return false, fmt.Errorf(wrapCompensateSaga, err)
	}

	return started, nil
}

func (sagaService *SagaService) compensate(
	processingContext context.Context,
	store PlotStore,
	sagaID string,
	reason domain.CompensationReason,
) (bool, error) {
	saga, started, err := store.FinishSaga(processingContext, SagaFinish{
		SagaID: sagaID,
		Status: domain.SagaCompensated,
		Reason: reason,
	})
	if err != nil || !started {
		return false, err
	}

	plot, err := store.LockPlot(processingContext, saga.PlotID)
	if err != nil {
		return false, err
	}

	compensated, eventType := saga.Compensate(plot, sagaService.now())
	if err := store.UpdatePlot(processingContext, compensated, plot.Version); err != nil {
		return false, err
	}

	if err := store.AppendEvent(
		processingContext,
		domain.NewPlotEvent(uuid.NewString(), eventType, compensated),
	); err != nil {
		return false, err
	}

	sagaService.logger.Warn(
		logSagaCompensated,
		logging.KeySagaID, saga.ID,
		logging.KeyPlotID, saga.PlotID,
		logging.KeySagaKind, saga.Kind,
		logging.KeyReason, reason,
	)

	return true, nil
}
