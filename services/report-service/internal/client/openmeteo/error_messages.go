package openmeteo

const (
	wrapCreateRequest  = "create open-meteo request: %w"
	wrapSendRequest    = "send open-meteo request: %w"
	wrapDecodeResponse = "%w: decode body: %w"
	formatStatusReason = "%w: status %d: %s"
	formatMissingValue = "%w: value %d is missing"
	formatValueCount   = "%w: %d values for %d points"
)
