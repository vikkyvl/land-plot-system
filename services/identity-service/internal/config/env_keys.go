package config

const (
	envConsulAddress       = "CONSUL_ADDRESS"
	envConsulTimeout       = "CONSUL_TIMEOUT"
	envDatabaseTimeout     = "DATABASE_TIMEOUT"
	envDatabaseURL         = "DATABASE_URL"
	envHTTPAddress         = "HTTP_ADDRESS"
	envMetricsAddress      = "METRICS_ADDRESS"
	envDemoFaultsEnabled   = "DEMO_FAULTS_ENABLED"
	envOTLPEndpoint        = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envHTTPShutdownTimeout = "HTTP_SHUTDOWN_TIMEOUT"
	envServiceAddress      = "SERVICE_ADDRESS"
	envServiceInstanceID   = "SERVICE_INSTANCE_ID"
	envOIDCAudience        = "OIDC_AUDIENCE"
	envOIDCIssuer          = "OIDC_ISSUER"
	envOIDCJWKSURL         = "OIDC_JWKS_URL"
	envOIDCTimeout         = "OIDC_TIMEOUT"
)
