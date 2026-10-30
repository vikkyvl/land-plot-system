package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	exchangeName     = "land.events"
	deadExchangeName = "land.events.dead"
	queueName        = "report.plot-events"
	deadQueueName    = "report.plot-events.dead"
	allRoutingKeys   = "#"
)

func declareTopology(channel *amqp.Channel) error {
	for _, exchange := range []string{exchangeName, deadExchangeName} {
		if err := channel.ExchangeDeclare(exchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
			return fmt.Errorf(wrapDeclareExchange, exchange, err)
		}
	}

	queueArguments := amqp.Table{"x-dead-letter-exchange": deadExchangeName}
	if _, err := channel.QueueDeclare(queueName, true, false, false, false, queueArguments); err != nil {
		return fmt.Errorf(wrapDeclareQueue, queueName, err)
	}

	if _, err := channel.QueueDeclare(deadQueueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf(wrapDeclareQueue, deadQueueName, err)
	}

	bindings := []struct{ queue, routingKey, exchange string }{
		{queueName, string(domain.PlotCreated), exchangeName},
		{queueName, string(domain.PlotUpdated), exchangeName},
		{deadQueueName, allRoutingKeys, deadExchangeName},
	}
	for _, binding := range bindings {
		if err := channel.QueueBind(binding.queue, binding.routingKey, binding.exchange, false, nil); err != nil {
			return fmt.Errorf(wrapBindTo, binding.queue, binding.routingKey, err)
		}
	}

	return nil
}
