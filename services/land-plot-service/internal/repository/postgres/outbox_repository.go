package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (repository *OutboxRepository) PublishPending(
	requestContext context.Context,
	batch service.OutboxBatch,
	publish service.PublishFunc,
) (service.OutboxResult, error) {
	transaction, err := repository.pool.Begin(requestContext)
	if err != nil {
		return service.OutboxResult{}, fmt.Errorf(wrapBeginOutboxTransaction, err)
	}
	defer func() { _ = transaction.Rollback(requestContext) }()

	messages, err := claimPending(requestContext, transaction, batch.Limit)
	if err != nil {
		return service.OutboxResult{}, err
	}

	result, err := publishInOrder(requestContext, transaction, messages, outboxPublishing{
		publish:     publish,
		maxAttempts: batch.MaxAttempts,
	})
	if err != nil {
		return service.OutboxResult{}, err
	}

	if err := transaction.Commit(requestContext); err != nil {
		return service.OutboxResult{}, fmt.Errorf(wrapCommitOutboxTransaction, err)
	}

	return result, nil
}

func (repository *OutboxRepository) DeletePublishedBefore(
	requestContext context.Context,
	age time.Duration,
) (int64, error) {
	command, err := repository.pool.Exec(requestContext, `
		DELETE FROM outbox_events
		WHERE published_at < now() - make_interval(secs => $1)`,
		age.Seconds(),
	)
	if err != nil {
		return 0, fmt.Errorf(wrapDeletePublishedOutboxEvents, err)
	}

	return command.RowsAffected(), nil
}

func claimPending(requestContext context.Context, transaction pgx.Tx, limit int) ([]domain.OutboxMessage, error) {
	rows, err := transaction.Query(requestContext, `
		SELECT id, routing_key, payload FROM outbox_events
		WHERE published_at IS NULL AND failed_at IS NULL
		ORDER BY position
		LIMIT $1
		FOR UPDATE SKIP LOCKED`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf(wrapClaimOutboxEvents, err)
	}

	messages, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.OutboxMessage, error) {
		var message domain.OutboxMessage

		err := row.Scan(&message.ID, &message.RoutingKey, &message.Payload)

		return message, err
	})
	if err != nil {
		return nil, fmt.Errorf(wrapClaimOutboxEvents, err)
	}

	return messages, nil
}

type outboxPublishing struct {
	publish     service.PublishFunc
	maxAttempts int
}

func publishInOrder(
	requestContext context.Context,
	transaction pgx.Tx,
	messages []domain.OutboxMessage,
	publishing outboxPublishing,
) (service.OutboxResult, error) {
	for published, message := range messages {
		if publishErr := publishing.publish(requestContext, message); publishErr != nil {
			abandoned, err := recordFailure(requestContext, transaction, message, outboxFailure{
				cause:       publishErr,
				maxAttempts: publishing.maxAttempts,
			})
			if err != nil {
				return service.OutboxResult{}, err
			}

			return service.OutboxResult{
				Published:  published,
				PublishErr: fmt.Errorf(wrapPublishOutboxEvent, publishErr),
				Abandoned:  abandoned,
			}, nil
		}

		if _, err := transaction.Exec(requestContext,
			`UPDATE outbox_events SET published_at = now() WHERE id = $1`, message.ID,
		); err != nil {
			return service.OutboxResult{}, fmt.Errorf(wrapMarkOutboxEventPublished, err)
		}
	}

	return service.OutboxResult{Published: len(messages)}, nil
}

type outboxFailure struct {
	cause       error
	maxAttempts int
}

func recordFailure(
	requestContext context.Context,
	transaction pgx.Tx,
	message domain.OutboxMessage,
	failure outboxFailure,
) (*service.AbandonedMessage, error) {
	lastError := failure.cause.Error()

	if !errors.Is(failure.cause, domain.ErrMessageRejected) {
		if _, err := transaction.Exec(requestContext,
			`UPDATE outbox_events SET last_error = $2 WHERE id = $1`, message.ID, lastError,
		); err != nil {
			return nil, fmt.Errorf(wrapRecordOutboxFailure, err)
		}

		return nil, nil
	}

	var (
		attempts  int
		abandoned bool
	)

	if err := transaction.QueryRow(requestContext, `
		UPDATE outbox_events
		SET attempts = attempts + 1,
			last_error = $2,
			failed_at = CASE WHEN attempts + 1 >= $3 THEN now() END
		WHERE id = $1
		RETURNING attempts, failed_at IS NOT NULL`,
		message.ID, lastError, failure.maxAttempts,
	).Scan(&attempts, &abandoned); err != nil {
		return nil, fmt.Errorf(wrapRecordOutboxFailure, err)
	}

	if !abandoned {
		return nil, nil
	}

	return &service.AbandonedMessage{
		ID:         message.ID,
		RoutingKey: message.RoutingKey,
		Attempts:   attempts,
		LastError:  lastError,
	}, nil
}
