package rabbitmq

const (
	wrapAcknowledgeSagaMessage       = "acknowledge saga message: %w"
	wrapBindTo                       = "bind %s to %s: %w"
	wrapBrokerRejectedMessage        = "%w: broker returned nack"
	wrapCloseRabbitMQChannel         = "close RabbitMQ channel: %w"
	wrapCloseRabbitMQConnection      = "close RabbitMQ connection: %w"
	wrapConfigureConsumerPrefetch    = "configure consumer prefetch: %w"
	wrapConnectToRabbitMQ            = "connect to RabbitMQ: %w"
	wrapDecodeJSON                   = "%w: decode JSON: %w"
	wrapDeclareExchange              = "declare exchange %s: %w"
	wrapDeclareQueue                 = "declare queue %s: %w"
	wrapEnablePublisherConfirms      = "enable publisher confirms: %w"
	wrapMessageNotRouted             = "%w: routing key %s"
	wrapMessageUnroutable            = "%w: no queue is bound for the routing key"
	wrapOpenRabbitMQChannel          = "open RabbitMQ channel: %w"
	wrapPublishMessage               = "publish message: %w"
	wrapReconnectToRabbitMQ          = "reconnect to RabbitMQ: %w"
	wrapRejectSagaMessage            = "reject saga message: %w"
	wrapRequeueSagaMessage           = "requeue saga message: %w"
	wrapStartSagaConsumer            = "start saga consumer: %w"
	wrapWaitForPublisherConfirmation = "wait for publisher confirmation: %w"
)
