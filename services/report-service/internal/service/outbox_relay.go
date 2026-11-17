package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
)

const outboxBatchSize = 50

type OutboxRelayConfig struct {
	PollInterval    time.Duration
	MaxAttempts     int
	CleanupInterval time.Duration
	Retention       time.Duration
}

type OutboxRelay struct {
	repository  OutboxRepository
	publisher   MessagePublisher
	relayConfig OutboxRelayConfig
	logger      *slog.Logger
}

func NewOutboxRelay(
	repository OutboxRepository,
	publisher MessagePublisher,
	relayConfig OutboxRelayConfig,
	logger *slog.Logger,
) *OutboxRelay {
	return &OutboxRelay{repository: repository, publisher: publisher, relayConfig: relayConfig, logger: logger}
}

func (relay *OutboxRelay) Run(runContext context.Context) {
	pollTicker := time.NewTicker(relay.relayConfig.PollInterval)
	defer pollTicker.Stop()

	cleanupTicker := time.NewTicker(relay.relayConfig.CleanupInterval)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-runContext.Done():
			return
		case <-pollTicker.C:
			relay.relayOnce(runContext)
		case <-cleanupTicker.C:
			relay.cleanUp(runContext)
		}
	}
}

func (relay *OutboxRelay) relayOnce(runContext context.Context) {
	result, err := relay.repository.PublishPending(
		runContext,
		OutboxBatch{Limit: outboxBatchSize, MaxAttempts: relay.relayConfig.MaxAttempts},
		relay.publisher.Publish,
	)
	if err != nil {
		relay.logger.Error(logRelayOutboxMessages, logging.KeyError, err)
		return
	}

	if result.Published > 0 {
		relay.logger.Info(logOutboxMessagesPublished, logging.KeyCount, result.Published)
	}

	if result.Abandoned != nil {
		relay.logger.Error(
			logOutboxMessageAbandoned,
			logging.KeyMessageID, result.Abandoned.ID,
			logging.KeyRoutingKey, result.Abandoned.RoutingKey,
			logging.KeyAttempts, result.Abandoned.Attempts,
			logging.KeyLastError, result.Abandoned.LastError,
		)

		return
	}

	if result.PublishErr != nil {
		relay.logger.Warn(logRelayOutboxMessages, logging.KeyError, result.PublishErr)
	}
}

func (relay *OutboxRelay) cleanUp(runContext context.Context) {
	deleted, err := relay.repository.DeletePublishedBefore(runContext, relay.relayConfig.Retention)
	if err != nil {
		relay.logger.Error(logDeletePublishedOutboxMessages, logging.KeyError, err)
		return
	}

	if deleted > 0 {
		relay.logger.Info(logPublishedOutboxMessagesDeleted, logging.KeyCount, deleted)
	}
}
