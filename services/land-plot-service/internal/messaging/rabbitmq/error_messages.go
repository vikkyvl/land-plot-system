package rabbitmq

const (
	wrapCloseRabbitMQChannel         = "close RabbitMQ channel: %w"
	wrapCloseRabbitMQConnection      = "close RabbitMQ connection: %w"
	wrapConnectToRabbitMQ            = "connect to RabbitMQ: %w"
	wrapDeclareEventExchange         = "declare event exchange: %w"
	wrapEnablePublisherConfirms      = "enable publisher confirms: %w"
	wrapEncodePlotEvent              = "encode plot event: %w"
	wrapOpenRabbitMQChannel          = "open RabbitMQ channel: %w"
	wrapPublishPlotEvent             = "publish plot event: %w"
	wrapReconnectToRabbitMQ          = "reconnect to RabbitMQ: %w"
	wrapWaitForPublisherConfirmation = "wait for publisher confirmation: %w"
)
