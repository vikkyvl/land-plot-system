package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const (
	testMaxAttempts = 3
	testRetention   = 2 * time.Hour
)

var errBrokerUnavailable = errors.New("broker unavailable")

type outboxRepositoryStub struct {
	pending      []domain.OutboxMessage
	published    []string
	batch        OutboxBatch
	abandoned    *AbandonedMessage
	deletedAge   time.Duration
	deleteCalled bool
}

func (stub *outboxRepositoryStub) PublishPending(
	publishContext context.Context,
	batch OutboxBatch,
	publish PublishFunc,
) (OutboxResult, error) {
	stub.batch = batch

	for _, message := range stub.pending {
		publishErr := publish(publishContext, message)
		if publishErr == nil {
			stub.published = append(stub.published, message.ID)
			continue
		}

		return OutboxResult{Published: len(stub.published), PublishErr: publishErr, Abandoned: stub.abandoned}, nil
	}

	return OutboxResult{Published: len(stub.published)}, nil
}

func (stub *outboxRepositoryStub) DeletePublishedBefore(_ context.Context, age time.Duration) (int64, error) {
	stub.deleteCalled = true
	stub.deletedAge = age

	return 0, nil
}

type messagePublisherStub struct {
	err error
}

func (stub messagePublisherStub) Publish(context.Context, domain.OutboxMessage) error {
	return stub.err
}

func newTestRelay(repository OutboxRepository, publisher MessagePublisher, logger *slog.Logger) *OutboxRelay {
	return NewOutboxRelay(
		repository,
		publisher,
		OutboxRelayConfig{
			PollInterval:    time.Second,
			MaxAttempts:     testMaxAttempts,
			CleanupInterval: time.Minute,
			Retention:       testRetention,
		},
		logger,
	)
}

func TestOutboxRelayPublishesPendingMessagesWithoutCleanup(t *testing.T) {
	repository := &outboxRepositoryStub{pending: []domain.OutboxMessage{{ID: "event-1"}, {ID: "event-2"}}}

	newTestRelay(repository, messagePublisherStub{}, discardLogger()).relayOnce(context.Background())

	if len(repository.published) != 2 || repository.deleteCalled {
		t.Fatalf("published = %v, delete called = %v", repository.published, repository.deleteCalled)
	}

	if repository.batch != (OutboxBatch{Limit: outboxBatchSize, MaxAttempts: testMaxAttempts}) {
		t.Fatalf("batch = %#v", repository.batch)
	}
}

func TestOutboxRelayCleanUpUsesConfiguredRetention(t *testing.T) {
	repository := &outboxRepositoryStub{}

	newTestRelay(repository, messagePublisherStub{}, discardLogger()).cleanUp(context.Background())

	if !repository.deleteCalled || repository.deletedAge != testRetention {
		t.Fatalf("delete called = %v, age = %v", repository.deleteCalled, repository.deletedAge)
	}
}

func TestOutboxRelayKeepsMessagesWhenBrokerIsUnavailable(t *testing.T) {
	repository := &outboxRepositoryStub{pending: []domain.OutboxMessage{{ID: "event-1"}}}
	publisher := messagePublisherStub{err: errBrokerUnavailable}

	newTestRelay(repository, publisher, discardLogger()).relayOnce(context.Background())

	if len(repository.published) != 0 || repository.deleteCalled {
		t.Fatalf("published = %v, delete called = %v", repository.published, repository.deleteCalled)
	}
}

func TestOutboxRelayLogsAbandonedMessage(t *testing.T) {
	repository := &outboxRepositoryStub{
		pending: []domain.OutboxMessage{{ID: "event-1", RoutingKey: "plot.created"}},
		abandoned: &AbandonedMessage{
			ID: "event-1", RoutingKey: "plot.created", Attempts: testMaxAttempts, LastError: "no queue",
		},
	}

	var logs bytes.Buffer

	publisher := messagePublisherStub{err: domain.ErrMessageRejected}
	newTestRelay(repository, publisher, slog.New(slog.NewTextHandler(&logs, nil))).relayOnce(context.Background())

	for _, want := range []string{
		"level=ERROR",
		"message_id=event-1",
		"routing_key=plot.created",
		"attempts=3",
		`last_error="no queue"`,
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("logs = %q, want %q", logs.String(), want)
		}
	}
}
