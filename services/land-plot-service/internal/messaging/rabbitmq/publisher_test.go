package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

var (
	errSessionClosed = errors.New("channel/connection is not open")
	errDialFailed    = errors.New("dial failed")
	errPublishFailed = errors.New("publish failed")
)

type sessionStub struct {
	closed      bool
	publishErr  error
	published   []string
	closeCalled bool
}

func (stub *sessionStub) publish(_ context.Context, routingKey string, _ amqp.Publishing) error {
	if stub.publishErr != nil {
		return stub.publishErr
	}

	stub.published = append(stub.published, routingKey)

	return nil
}

func (stub *sessionStub) isClosed() bool { return stub.closed }

func (stub *sessionStub) close() error {
	stub.closeCalled = true
	return nil
}

type dialerStub struct {
	sessions []*sessionStub
	err      error
	calls    int
}

func (stub *dialerStub) dial() (brokerSession, error) {
	stub.calls++
	if stub.err != nil {
		return nil, stub.err
	}

	session := stub.sessions[0]
	stub.sessions = stub.sessions[1:]

	return session, nil
}

func newTestPublisher(session *sessionStub, dialer *dialerStub) *Publisher {
	return &Publisher{session: session, dial: dialer.dial, timeout: time.Second}
}

func testEvent() domain.PlotEvent {
	return domain.PlotEvent{ID: "event-1", Type: domain.PlotCreated, Plot: domain.Plot{ID: "plot-1"}}
}

func TestPublisherPublishesOnOpenSessionWithoutDialing(t *testing.T) {
	session := &sessionStub{}
	dialer := &dialerStub{}

	if err := newTestPublisher(session, dialer).Publish(context.Background(), testEvent()); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if len(session.published) != 1 || dialer.calls != 0 {
		t.Fatalf("published = %v, dial calls = %d", session.published, dialer.calls)
	}
}

func TestPublisherReconnectsAndRetriesOnceWhenSessionIsClosed(t *testing.T) {
	brokenSession := &sessionStub{closed: true, publishErr: errSessionClosed}
	freshSession := &sessionStub{}
	dialer := &dialerStub{sessions: []*sessionStub{freshSession}}
	publisher := newTestPublisher(brokenSession, dialer)

	if err := publisher.Publish(context.Background(), testEvent()); err != nil {
		t.Fatalf("Publish() error = %v, want success after reconnect", err)
	}

	if dialer.calls != 1 || !brokenSession.closeCalled {
		t.Fatalf("dial calls = %d, broken session closed = %v", dialer.calls, brokenSession.closeCalled)
	}

	if len(freshSession.published) != 1 || freshSession.published[0] != string(domain.PlotCreated) {
		t.Fatalf("fresh session published = %v", freshSession.published)
	}

	if publisher.session != freshSession {
		t.Fatal("publisher did not keep the fresh session for later events")
	}
}

func TestPublisherDoesNotReconnectWhenSessionIsOpen(t *testing.T) {
	session := &sessionStub{publishErr: errPublishFailed}
	dialer := &dialerStub{}

	err := newTestPublisher(session, dialer).Publish(context.Background(), testEvent())
	if !errors.Is(err, errPublishFailed) || dialer.calls != 0 {
		t.Fatalf("Publish() error = %v, dial calls = %d", err, dialer.calls)
	}
}

func TestPublisherReturnsErrorWhenReconnectFails(t *testing.T) {
	brokenSession := &sessionStub{closed: true, publishErr: errSessionClosed}
	dialer := &dialerStub{err: errDialFailed}
	publisher := newTestPublisher(brokenSession, dialer)

	err := publisher.Publish(context.Background(), testEvent())
	if !errors.Is(err, errDialFailed) || dialer.calls != 1 {
		t.Fatalf("Publish() error = %v, dial calls = %d", err, dialer.calls)
	}

	if publisher.session != brokenSession {
		t.Fatal("publisher replaced the session although reconnect failed")
	}
}

func TestPublisherRetriesOnlyOnce(t *testing.T) {
	brokenSession := &sessionStub{closed: true, publishErr: errSessionClosed}
	alsoBrokenSession := &sessionStub{closed: true, publishErr: errSessionClosed}
	dialer := &dialerStub{sessions: []*sessionStub{alsoBrokenSession}}

	err := newTestPublisher(brokenSession, dialer).Publish(context.Background(), testEvent())
	if !errors.Is(err, errSessionClosed) || dialer.calls != 1 {
		t.Fatalf("Publish() error = %v, dial calls = %d", err, dialer.calls)
	}
}
