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

const (
	expiredSagaBatchSize = 50
	operationSagaTimeout = "saga timeout"
)

type SagaService struct {
	repository PlotRepository
	metrics    SagaMetrics
	traces     TraceResumer
	logger     *slog.Logger
	now        func() time.Time
}

type compensation struct {
	saga   domain.Saga
	result domain.PlotEventType
}

func NewSagaService(
	repository PlotRepository,
	metrics SagaMetrics,
	traces TraceResumer,
	logger *slog.Logger,
) *SagaService {
	return &SagaService{repository: repository, metrics: metrics, traces: traces, logger: logger, now: time.Now}
}

func (sagaService *SagaService) Confirm(processingContext context.Context, sagaID string) error {
	var finished *domain.Saga

	err := sagaService.repository.InTransaction(processingContext, func(store PlotStore) error {
		finished = nil

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

		confirmed := plot.Confirmed()
		if err := store.UpdatePlot(processingContext, confirmed, plot.Version); err != nil {
			return err
		}

		if err := store.RecordVersion(processingContext, confirmed.ConfirmedVersion()); err != nil {
			return err
		}

		sagaService.logger.InfoContext(
			processingContext,
			logSagaCompleted,
			logging.KeySagaID, saga.ID,
			logging.KeyPlotID, saga.PlotID,
		)

		finished = &saga

		return nil
	})
	if err != nil {
		return fmt.Errorf(wrapConfirmSaga, err)
	}

	if finished != nil {
		sagaService.metrics.SagaConfirmed(finished.Kind)
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
	sagas, err := sagaService.repository.ExpiredSagas(processingContext, expiredSagaBatchSize)
	if err != nil {
		return 0, fmt.Errorf(wrapCompensateExpiredSagas, err)
	}

	var (
		compensated int
		failures    []error
	)

	for _, saga := range sagas {
		started, err := sagaService.compensateExpired(processingContext, saga)
		if err != nil {
			sagaService.logger.ErrorContext(
				processingContext,
				logCompensateExpiredSaga,
				logging.KeySagaID,
				saga.ID,
				logging.KeyError,
				err,
			)
			failures = append(failures, err)

			continue
		}

		if started {
			compensated++
		}
	}

	return compensated, errors.Join(failures...)
}

func (sagaService *SagaService) compensateExpired(parent context.Context, saga domain.ExpiredSaga) (bool, error) {
	resumed, end := sagaService.traces.Resume(parent, saga.TraceContext, operationSagaTimeout)
	defer end()

	return sagaService.compensateInTransaction(resumed, saga.ID, domain.ReasonTimeout)
}

func (sagaService *SagaService) compensateInTransaction(
	processingContext context.Context,
	sagaID string,
	reason domain.CompensationReason,
) (bool, error) {
	var done *compensation

	err := sagaService.repository.InTransaction(processingContext, func(store PlotStore) error {
		var err error

		done, err = sagaService.compensate(processingContext, store, sagaID, reason)

		return err
	})
	if err != nil {
		return false, fmt.Errorf(wrapCompensateSaga, err)
	}

	if done == nil {
		return false, nil
	}

	sagaService.metrics.SagaCompensated(done.saga.Kind, done.result, reason)

	return true, nil
}

func (sagaService *SagaService) compensate(
	processingContext context.Context,
	store PlotStore,
	sagaID string,
	reason domain.CompensationReason,
) (*compensation, error) {
	saga, started, err := store.FinishSaga(processingContext, SagaFinish{
		SagaID: sagaID,
		Status: domain.SagaCompensated,
		Reason: reason,
	})
	if err != nil || !started {
		return nil, err
	}

	plot, err := store.LockPlot(processingContext, saga.PlotID)
	if err != nil {
		return nil, err
	}

	compensated, eventType := saga.Compensate(plot, sagaService.now())
	if err := store.UpdatePlot(processingContext, compensated, plot.Version); err != nil {
		return nil, err
	}

	if compensated.Status == domain.StatusConfirmed {
		if err := store.RecordVersion(processingContext, compensated.ConfirmedVersion()); err != nil {
			return nil, err
		}
	}

	if err := store.AppendEvent(
		processingContext,
		domain.NewPlotEvent(uuid.NewString(), eventType, compensated),
	); err != nil {
		return nil, err
	}

	sagaService.logger.WarnContext(
		processingContext,
		logSagaCompensated,
		logging.KeySagaID, saga.ID,
		logging.KeyPlotID, saga.PlotID,
		logging.KeySagaKind, saga.Kind,
		logging.KeyReason, reason,
	)

	return &compensation{saga: saga, result: eventType}, nil
}
