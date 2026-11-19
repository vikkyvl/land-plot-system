package config

const (
	envCORSAllowedOrigins  = "CORS_ALLOWED_ORIGINS"
	envConsulAddress       = "CONSUL_ADDRESS"
	envConsulTimeout       = "CONSUL_TIMEOUT"
	envHTTPAddress         = "HTTP_ADDRESS"
	envHTTPShutdownTimeout = "HTTP_SHUTDOWN_TIMEOUT"
	envServiceAddress      = "SERVICE_ADDRESS"
	envServiceInstanceID   = "SERVICE_INSTANCE_ID"
	envUpstreamTimeout     = "UPSTREAM_TIMEOUT"
	envOIDCAudience        = "OIDC_AUDIENCE"
	envOIDCIssuer          = "OIDC_ISSUER"
	envOIDCJWKSURL         = "OIDC_JWKS_URL"
	envOIDCTimeout         = "OIDC_TIMEOUT"
	envPreviewRateLimit    = "PREVIEW_RATE_LIMIT_PER_MINUTE"
)
