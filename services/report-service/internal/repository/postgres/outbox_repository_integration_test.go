//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/service"
)

const testRoutingKey = "plot.created"

var errTestBrokerDown = errors.New("dial tcp: connection refused")

type outboxRow struct {
	attempts  int
	published bool
	failed    bool
	lastError *string
}

func insertOutboxMessages(t *testing.T, pool *pgxpool.Pool, count int) []string {
	t.Helper()

	ids := make([]string, count)
	for i := range ids {
		ids[i] = uuid.NewString()
		if _, err := pool.Exec(context.Background(),
			`INSERT INTO outbox_events (id, routing_key, payload) VALUES ($1, $2, '{}')`, ids[i], testRoutingKey,
		); err != nil {
			t.Fatalf("insert outbox message: %v", err)
		}
	}

	return ids
}

func readOutboxRow(t *testing.T, pool *pgxpool.Pool, id string) outboxRow {
	t.Helper()

	var row outboxRow
	if err := pool.QueryRow(context.Background(), `
		SELECT attempts, published_at IS NOT NULL, failed_at IS NOT NULL, last_error
		FROM outbox_events WHERE id = $1`, id,
	).Scan(&row.attempts, &row.published, &row.failed, &row.lastError); err != nil {
		t.Fatalf("read outbox row %s: %v", id, err)
	}

	return row
}

type publishRecorder struct {
	published []string
	failures  map[string]error
}

func (recorder *publishRecorder) publish(_ context.Context, message domain.OutboxMessage) error {
	if err := recorder.failures[message.ID]; err != nil {
		return err
	}

	recorder.published = append(recorder.published, message.ID)

	return nil
}

func publishPending(
	t *testing.T,
	repository *OutboxRepository,
	recorder *publishRecorder,
	maxAttempts int,
) service.OutboxResult {
	t.Helper()

	result, err := repository.PublishPending(
		context.Background(),
		service.OutboxBatch{Limit: 50, MaxAttempts: maxAttempts},
		recorder.publish,
	)
	if err != nil {
		t.Fatalf("PublishPending() error = %v", err)
	}

	return result
}

func TestOutboxPublishesInInsertionOrder(t *testing.T) {
	pool := openTestPool(t)
	ids := insertOutboxMessages(t, pool, 3)
	recorder := &publishRecorder{}

	result := publishPending(t, NewOutboxRepository(pool), recorder, 10)

	if result.Published != 3 || !slices.Equal(recorder.published, ids) {
		t.Fatalf("published = %v (%d), want %v", recorder.published, result.Published, ids)
	}

	for _, id := range ids {
		if !readOutboxRow(t, pool, id).published {
			t.Fatalf("message %s has no published_at", id)
		}
	}
}

func TestOutboxRejectedMessageIsAbandonedAfterMaxAttemptsAndUnblocksQueue(t *testing.T) {
	pool := openTestPool(t)
	ids := insertOutboxMessages(t, pool, 2)
	repository := NewOutboxRepository(pool)
	rejection := fmt.Errorf("%w: no queue is bound", domain.ErrMessageRejected)
	recorder := &publishRecorder{failures: map[string]error{ids[0]: rejection}}

	first := publishPending(t, repository, recorder, 2)
	if row := readOutboxRow(t, pool, ids[0]); row.attempts != 1 || row.failed || first.Abandoned != nil {
		t.Fatalf("after first attempt row = %#v, abandoned = %#v", row, first.Abandoned)
	}

	second := publishPending(t, repository, recorder, 2)
	if row := readOutboxRow(t, pool, ids[0]); row.attempts != 2 || !row.failed {
		t.Fatalf("after last attempt row = %#v", row)
	}

	if second.Abandoned == nil || second.Abandoned.ID != ids[0] || second.Abandoned.Attempts != 2 {
		t.Fatalf("abandoned = %#v", second.Abandoned)
	}

	third := publishPending(t, repository, recorder, 2)
	if third.Published != 1 || !slices.Equal(recorder.published, ids[1:]) {
		t.Fatalf("after abandon published = %v (%d), want %v", recorder.published, third.Published, ids[1:])
	}

	if row := readOutboxRow(t, pool, ids[0]); row.attempts != 2 || row.published {
		t.Fatalf("abandoned row was claimed again: %#v", row)
	}
}

func TestOutboxUnavailableBrokerKeepsAttempts(t *testing.T) {
	pool := openTestPool(t)
	ids := insertOutboxMessages(t, pool, 1)
	recorder := &publishRecorder{failures: map[string]error{ids[0]: errTestBrokerDown}}

	for range 3 {
		publishPending(t, NewOutboxRepository(pool), recorder, 2)
	}

	row := readOutboxRow(t, pool, ids[0])
	if row.attempts != 0 || row.failed || row.published {
		t.Fatalf("row = %#v, want no attempts while the broker is down", row)
	}

	if row.lastError == nil || *row.lastError != errTestBrokerDown.Error() {
		t.Fatalf("last error = %v, want %q", row.lastError, errTestBrokerDown)
	}
}

func TestOutboxCleanupDeletesOnlyOldPublishedMessages(t *testing.T) {
	pool := openTestPool(t)
	ids := insertOutboxMessages(t, pool, 3)

	if _, err := pool.Exec(context.Background(), `
		UPDATE outbox_events SET published_at = now() - interval '2 hours' WHERE id = $1;`, ids[0]); err != nil {
		t.Fatalf("age published message: %v", err)
	}

	if _, err := pool.Exec(context.Background(), `
		UPDATE outbox_events SET failed_at = now() - interval '2 hours' WHERE id = $1;`, ids[1]); err != nil {
		t.Fatalf("age failed message: %v", err)
	}

	deleted, err := NewOutboxRepository(pool).DeletePublishedBefore(context.Background(), time.Hour)
	if err != nil || deleted != 1 {
		t.Fatalf("DeletePublishedBefore() = %d, %v, want 1", deleted, err)
	}

	var remaining []string

	rows, err := pool.Query(context.Background(), `SELECT id::text FROM outbox_events ORDER BY position`)
	if err != nil {
		t.Fatalf("query remaining: %v", err)
	}

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan remaining: %v", err)
		}

		remaining = append(remaining, id)
	}

	if !slices.Equal(remaining, ids[1:]) {
		t.Fatalf("remaining = %v, want %v", remaining, ids[1:])
	}
}
