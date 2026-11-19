package rabbitmq

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	HeaderCreatedAt = "x-created-at-ms"

	OutcomeAcknowledged = "ack"
	OutcomeRequeued     = "retry"
	OutcomeRejected     = "reject"
)

type DeliveryObserver interface {
	DeliveryProcessed(queue, outcome string, createdAt time.Time)
}

func createdAt(delivery amqp.Delivery) time.Time {
	if milliseconds, ok := delivery.Headers[HeaderCreatedAt].(int64); ok {
		return time.UnixMilli(milliseconds)
	}

	return delivery.Timestamp
}
