package tools

const (
	codeInvalidArguments   = "invalid_arguments"
	codeUnknownTool        = "unknown_tool"
	codeGatewayUnavailable = "gateway_unavailable"
	codeVersionMismatch    = "version_mismatch"

	formatMissingArgument        = "%s is required"
	formatMustBeUUID             = "%s must be a UUID"
	formatPointMissingCoordinate = "point %d must contain lat and lon"
	formatPointOutOfRange        = "point %d is out of range: latitude must be from -90 to 90, longitude from -180 to 180"
	formatToolNotAllowed         = "tool %s is not available"
	wrapDecodeArguments          = "arguments must be a JSON object with known fields only: %w"

	messageGatewayUnavailable = "the land plot system is temporarily unavailable; try again later"
	messageVersionMismatch    = "the plot version is lower than expected_version; " +
		"pass the version returned by create_plot or update_plot"
	messageTruncated = "the result was too long and was truncated"
)
