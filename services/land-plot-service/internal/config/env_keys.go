package config

const (
	envConsulAddress            = "CONSUL_ADDRESS"
	envConsulTimeout            = "CONSUL_TIMEOUT"
	envDatabaseTimeout          = "DATABASE_TIMEOUT"
	envDatabaseURL              = "DATABASE_URL"
	envHTTPAddress              = "HTTP_ADDRESS"
	envHTTPShutdownTimeout      = "HTTP_SHUTDOWN_TIMEOUT"
	envJWTSecret                = "JWT_SECRET"
	envMessageProcessingTimeout = "MESSAGE_PROCESSING_TIMEOUT"
	envOutboxCleanupInterval    = "OUTBOX_CLEANUP_INTERVAL"
	envOutboxMaxAttempts        = "OUTBOX_MAX_ATTEMPTS"
	envOutboxPollInterval       = "OUTBOX_POLL_INTERVAL"
	envOutboxRetention          = "OUTBOX_RETENTION"
	envRabbitMQTimeout          = "RABBITMQ_TIMEOUT"
	envRabbitMQURL              = "RABBITMQ_URL"
	envSagaTimeout              = "SAGA_TIMEOUT"
	envServiceAddress           = "SERVICE_ADDRESS"
	envServiceInstanceID        = "SERVICE_INSTANCE_ID"
)
