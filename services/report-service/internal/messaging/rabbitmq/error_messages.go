package rabbitmq

const (
	wrapAcknowledgePlotEvent            = "acknowledge plot event: %w"
	wrapBindTo                          = "bind %s to %s: %w"
	wrapBodyMustContainOneJSONObject    = "%w: body must contain one JSON object"
	wrapCloseRabbitMQChannel            = "close RabbitMQ channel: %w"
	wrapCloseRabbitMQConnection         = "close RabbitMQ connection: %w"
	wrapConfigureConsumerPrefetch       = "configure consumer prefetch: %w"
	wrapConnectToRabbitMQ               = "connect to RabbitMQ: %w"
	wrapDeclareExchange                 = "declare exchange %s: %w"
	wrapDeclareQueue                    = "declare queue %s: %w"
	wrapDecodeJSON                      = "%w: decode JSON: %w"
	wrapEventTypeDoesNotMatchRoutingKey = "%w: event type does not match routing key"
	wrapInvalidBodySize                 = "%w: invalid body size"
	wrapOpenRabbitMQChannel             = "open RabbitMQ channel: %w"
	wrapRejectPlotEvent                 = "reject plot event: %w"
	wrapStartPlotEventConsumer          = "start plot event consumer: %w"
)
