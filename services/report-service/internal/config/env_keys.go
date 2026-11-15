package config

const (
	envConsulAddress            = "CONSUL_ADDRESS"
	envConsulTimeout            = "CONSUL_TIMEOUT"
	envDatabaseTimeout          = "DATABASE_TIMEOUT"
	envDatabaseURL              = "DATABASE_URL"
	envHTTPAddress              = "HTTP_ADDRESS"
	envHTTPShutdownTimeout      = "HTTP_SHUTDOWN_TIMEOUT"
	envJWTSecret                = "JWT_SECRET"
	envLandPlotServiceName      = "LAND_PLOT_SERVICE_NAME"
	envMessageProcessingTimeout = "MESSAGE_PROCESSING_TIMEOUT"
	envOutboxCleanupInterval    = "OUTBOX_CLEANUP_INTERVAL"
	envOutboxMaxAttempts        = "OUTBOX_MAX_ATTEMPTS"
	envOutboxPollInterval       = "OUTBOX_POLL_INTERVAL"
	envOutboxRetention          = "OUTBOX_RETENTION"
	envRabbitMQPrefetch         = "RABBITMQ_PREFETCH"
	envRabbitMQTimeout          = "RABBITMQ_TIMEOUT"
	envRabbitMQURL              = "RABBITMQ_URL"
	envReportFailNamePrefix     = "REPORT_FAIL_NAME_PREFIX"
	envServiceAddress           = "SERVICE_ADDRESS"
	envServiceInstanceID        = "SERVICE_INSTANCE_ID"
	envUpstreamTimeout          = "UPSTREAM_TIMEOUT"
)
