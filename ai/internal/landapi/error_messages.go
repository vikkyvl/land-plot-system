package landapi

const (
	formatAPIError          = "gateway returned %d %s: %s"
	wrapBuildRequest        = "build request %s %s: %w"
	wrapDecodeResponse      = "decode response of %s %s: %w"
	wrapEncodeRequest       = "encode request %s %s: %w"
	wrapReadResponse        = "read response of %s %s: %w"
	wrapSendRequest         = "send request %s %s: %w"
	wrapBuildGrantRequest   = "build token request: %w"
	wrapDecodeGrantResponse = "decode token response: %w"
	wrapReadGrantResponse   = "read token response: %w"
	wrapRequestToken        = "get an access token for the AI client: %w"
	wrapSendGrantRequest    = "send token request: %w"
	unexpectedStatusText    = "unexpected response"
)
