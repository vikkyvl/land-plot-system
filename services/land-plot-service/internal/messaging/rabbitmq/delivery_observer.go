package rabbitmq

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	HeaderCreatedAt = "x-created-at-ms"
	HeaderRequestID = "x-request-id"

	OutcomeAcknowledged = "ack"
	OutcomeRequeued     = "retry"
	OutcomeRejected     = "reject"
	OutcomePublished    = "published"
	OutcomeNotPublished = "not_published"
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

func requestID(delivery amqp.Delivery) string {
	value, _ := delivery.Headers[HeaderRequestID].(string)

	return value
}
