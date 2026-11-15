package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

type plotSummaryRepositoryStub struct {
	event domain.PlotEvent
	reply domain.SummaryApplied
	err   error
}

func (stub *plotSummaryRepositoryStub) Apply(
	_ context.Context,
	event domain.PlotEvent,
	reply domain.SummaryApplied,
) error {
	stub.event = event
	stub.reply = reply

	return stub.err
}

func TestPlotProjectionAppliesEvent(t *testing.T) {
	repository := &plotSummaryRepositoryStub{}
	event := domain.PlotEvent{ID: "event-1", Type: domain.PlotCreated, PlotID: "plot-1", Version: 1}

	if err := NewPlotProjection(repository).Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if repository.event.ID != event.ID {
		t.Fatalf("repository event = %#v", repository.event)
	}

	reply := repository.reply
	if _, err := uuid.Parse(reply.ID); err != nil || reply.AppliedEventID != event.ID ||
		reply.PlotID != event.PlotID || reply.PlotVersion != event.Version {
		t.Fatalf("reply = %#v", reply)
	}
}

func TestPlotProjectionWrapsRepositoryError(t *testing.T) {
	want := errors.New("database unavailable")

	err := NewPlotProjection(&plotSummaryRepositoryStub{err: want}).Handle(context.Background(), domain.PlotEvent{})
	if !errors.Is(err, want) {
		t.Fatalf("Handle() error = %v, want wrapped repository error", err)
	}
}
