package landapi

const (
	formatAPIError       = "gateway returned %d %s: %s"
	wrapBuildRequest     = "build request %s %s: %w"
	wrapDecodeResponse   = "decode response of %s %s: %w"
	wrapEncodeRequest    = "encode request %s %s: %w"
	wrapLogin            = "log in as the AI user: %w"
	wrapReadResponse     = "read response of %s %s: %w"
	wrapRegister         = "register the AI user: %w"
	wrapSendRequest      = "send request %s %s: %w"
	unexpectedStatusText = "unexpected response"
)
