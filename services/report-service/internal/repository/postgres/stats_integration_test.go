//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

func TestOutboxBacklogCountsPendingAndFailedRows(t *testing.T) {
	pool := openTestPool(t)
	repository := NewOutboxRepository(pool)
	ids := insertOutboxMessages(t, pool, 3)

	if _, err := pool.Exec(context.Background(),
		`UPDATE outbox_events SET created_at = now() - interval '40 seconds' WHERE id = $1`, ids[0],
	); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(
		context.Background(),
		`UPDATE outbox_events SET failed_at = now() WHERE id = $1`,
		ids[1],
	); err != nil {
		t.Fatal(err)
	}

	backlog, err := repository.Backlog(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if backlog.Pending != 2 || backlog.Failed != 1 || backlog.OldestAge < 39*time.Second ||
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

func TestReportCountsByPlotState(t *testing.T) {
	pool := openTestPool(t)
	ownerID := uuid.NewString()
	storeReport(t, pool, uuid.NewString(), ownerID)
	storeReport(t, pool, uuid.NewString(), ownerID)
	deleted := storeReport(t, pool, uuid.NewString(), ownerID)

	if _, err := pool.Exec(
		context.Background(),
		`UPDATE reports SET plot_deleted_at = now() WHERE id = $1`,
		deleted,
	); err != nil {
		t.Fatal(err)
	}

	counts, err := NewReportRepository(pool).Counts(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if counts != (domain.ReportCounts{ActivePlots: 2, DeletedPlots: 1}) {
		t.Fatalf("Counts() = %+v", counts)
	}
}
