package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
)

const sagaCheckInterval = 5 * time.Second

type SagaTimeoutWatcher struct {
	sagas  *SagaService
	logger *slog.Logger
}

func NewSagaTimeoutWatcher(sagas *SagaService, logger *slog.Logger) *SagaTimeoutWatcher {
	return &SagaTimeoutWatcher{sagas: sagas, logger: logger}
}

func (watcher *SagaTimeoutWatcher) Run(runContext context.Context) {
	runEvery(runContext, sagaCheckInterval, watcher.checkOnce)
}

func (watcher *SagaTimeoutWatcher) checkOnce(runContext context.Context) {
	compensated, err := watcher.sagas.CompensateExpired(runContext)
	if err != nil {
		watcher.logger.Error(logCompensateExpiredSagas, logging.KeyCount, compensated, logging.KeyError, err)
	}
}

func runEvery(runContext context.Context, interval time.Duration, task func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-runContext.Done():
			return
		case <-ticker.C:
			task(runContext)
		}
	}
}
