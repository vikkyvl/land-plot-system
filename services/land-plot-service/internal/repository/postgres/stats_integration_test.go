//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/logging"
	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/service"
)

func TestOutboxBacklogCountsPendingAndFailedRows(t *testing.T) {
	pool := openTestPool(t)
	repository := NewOutboxRepository(pool)

	empty, err := repository.Backlog(context.Background())
	if err != nil || empty != (domain.OutboxBacklog{}) {
		t.Fatalf("Backlog() on an empty outbox = %+v, %v", empty, err)
	}

	ids := insertOutboxMessages(t, pool, 4)
	statements := []string{
		`UPDATE outbox_events SET created_at = now() - interval '90 seconds' WHERE id = $1`,
		`UPDATE outbox_events SET published_at = now(), created_at = now() - interval '1 hour' WHERE id = $1`,
		`UPDATE outbox_events SET failed_at = now(), created_at = now() - interval '2 hours' WHERE id = $1`,
	}

	for index, statement := range statements {
		if _, err := pool.Exec(context.Background(), statement, ids[index]); err != nil {
			t.Fatal(err)
		}
	}

	backlog, err := repository.Backlog(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if backlog.Pending != 2 || backlog.Failed != 1 || backlog.OldestAge < 89*time.Second ||
		backlog.OldestAge > 5*time.Minute {
		t.Fatalf("Backlog() = %+v", backlog)
	}
}

func TestClaimedOutboxMessagesCarryTheirCreationTime(t *testing.T) {
	pool := openTestPool(t)
	insertOutboxMessages(t, pool, 1)

	recorder := &publishRecorder{}

	publishPending(t, NewOutboxRepository(pool), recorder, 1)

	if len(recorder.messages) != 1 || time.Since(recorder.messages[0].CreatedAt) > time.Minute {
		t.Fatalf("published messages = %+v", recorder.messages)
	}
}

func TestPlotCountsByState(t *testing.T) {
	pool := openTestPool(t)
	repository := NewPlotRepository(pool)
	ownerID := uuid.NewString()

	confirmed := storeConfirmedPlot(t, repository, ownerID)
	storeConfirmedPlot(t, repository, ownerID)
	deleted := storeConfirmedPlot(t, repository, ownerID)
	pending := storeConfirmedPlot(t, repository, ownerID)

	if _, err := pool.Exec(
		context.Background(),
		`UPDATE plots SET deleted_at = now() WHERE id = $1`,
		deleted.ID,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(
		context.Background(),
		`UPDATE plots SET status = 'PENDING' WHERE id = $1`,
		pending.ID,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(
		context.Background(),
		`UPDATE plots SET status = 'CANCELLED' WHERE id = $1`,
		confirmed.ID,
	); err != nil {
		t.Fatal(err)
	}

	counts, err := repository.Counts(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	want := domain.PlotCounts{Pending: 1, Confirmed: 1, Cancelled: 1, Deleted: 1}
	if counts != want {
		t.Fatalf("Counts() = %+v, want %+v", counts, want)
	}
}

func TestOutboxKeepsTheRequestIDOfTheEvent(t *testing.T) {
	pool := openTestPool(t)
	repository := NewPlotRepository(pool)
	plot := storeConfirmedPlot(t, repository, uuid.NewString())
	requestContext := logging.WithRequestID(context.Background(), "req-42")

	err := repository.InTransaction(requestContext, func(store service.PlotStore) error {
		return store.AppendEvent(requestContext, domain.NewPlotEvent(uuid.NewString(), domain.PlotUpdated, plot))
	})
	if err != nil {
		t.Fatal(err)
	}

	recorder := &publishRecorder{}
	publishPending(t, NewOutboxRepository(pool), recorder, 1)

	if len(recorder.messages) != 1 || recorder.messages[0].RequestID != "req-42" {
		t.Fatalf("published messages = %+v", recorder.messages)
	}
}
