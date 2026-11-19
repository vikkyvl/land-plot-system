//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
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

func TestReplyKeepsTheRequestIDOfTheEvent(t *testing.T) {
	pool := openTestPool(t)
	event := domain.PlotEvent{
		ID: uuid.NewString(), Type: domain.PlotCreated, OccurredAt: time.Now(),
		PlotID: uuid.NewString(), OwnerID: uuid.NewString(), Name: "Plot", AreaM2: 100, Version: 1,
	}
	reply := domain.NewSummaryApplied(uuid.NewString(), event, time.Now())

	err := NewPlotSummaryRepository(pool, "").Apply(logging.WithRequestID(context.Background(), "req-42"), event, reply)
	if err != nil {
		t.Fatal(err)
	}

	recorder := &publishRecorder{}
	publishPending(t, NewOutboxRepository(pool), recorder, 1)

	if len(recorder.messages) != 1 || recorder.messages[0].RequestID != "req-42" {
		t.Fatalf("published messages = %+v", recorder.messages)
	}
}

func spanContext(t *testing.T) context.Context {
	t.Helper()

	previous := otel.GetTextMapPropagator()

	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previous) })

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")

	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	}))
}

const storedTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

func TestReplyKeepsTheTraceContextOfTheConsumer(t *testing.T) {
	pool := openTestPool(t)
	event := domain.PlotEvent{
		ID: uuid.NewString(), Type: domain.PlotCreated, OccurredAt: time.Now(),
		PlotID: uuid.NewString(), OwnerID: uuid.NewString(), Name: "Plot", AreaM2: 100, Version: 1,
	}

	err := NewPlotSummaryRepository(pool, "").
		Apply(spanContext(t), event, domain.NewSummaryApplied(uuid.NewString(), event, time.Now()))
	if err != nil {
		t.Fatal(err)
	}

	recorder := &publishRecorder{}
	publishPending(t, NewOutboxRepository(pool), recorder, 1)

	if len(recorder.messages) != 1 || recorder.messages[0].TraceContext["traceparent"] != storedTraceparent {
		t.Fatalf("published messages = %+v", recorder.messages)
	}
}
