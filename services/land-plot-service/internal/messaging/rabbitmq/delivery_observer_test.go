package rabbitmq

import (
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

func TestHeadersCarryTheRequestIDOnlyWhenKnown(t *testing.T) {
	created := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)

	withID := messageHeaders(domain.OutboxMessage{CreatedAt: created, RequestID: "req-42"})
	if withID[HeaderRequestID] != "req-42" || withID[HeaderCreatedAt] != created.UnixMilli() {
		t.Fatalf("headers = %v", withID)
	}

	if _, found := messageHeaders(domain.OutboxMessage{CreatedAt: created})[HeaderRequestID]; found {
		t.Fatal("empty request ID was sent")
	}
}

func TestRequestIDIsReadFromTheHeader(t *testing.T) {
	if got := requestID(amqp.Delivery{Headers: amqp.Table{HeaderRequestID: "req-42"}}); got != "req-42" {
		t.Fatalf("requestID() = %q", got)
	}

	if got := requestID(amqp.Delivery{Headers: amqp.Table{HeaderRequestID: 42}}); got != "" {
		t.Fatalf("requestID() of a number = %q", got)
	}
}
