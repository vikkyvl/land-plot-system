package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/land-plot-service/internal/domain"
)

const (
	exchangeName           = "land.events"
	deadExchangeName       = "land.events.dead"
	sagaQueueName          = "land-plot.saga-events"
	summaryAppliedEvent    = "plot.summary.applied"
	contentTypeJSON        = "application/json"
	summaryAppliedSchemaV1 = 1
)

func declareSagaTopology(channel *amqp.Channel) error {
	for _, exchange := range []string{exchangeName, deadExchangeName} {
		if err := declareExchange(channel, exchange); err != nil {
			return err
		}
	}

	if _, err := channel.QueueDeclare(sagaQueueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf(wrapDeclareQueue, sagaQueueName, err)
	}

	bindings := []struct{ routingKey, exchange string }{
		{summaryAppliedEvent, exchangeName},
		{string(domain.PlotCreated), deadExchangeName},
		{string(domain.PlotUpdated), deadExchangeName},
	}
	for _, binding := range bindings {
		if err := channel.QueueBind(sagaQueueName, binding.routingKey, binding.exchange, false, nil); err != nil {
			return fmt.Errorf(wrapBindTo, sagaQueueName, binding.routingKey, err)
		}
	}

	return nil
}
