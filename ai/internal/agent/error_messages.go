package agent

const (
	formatToolNotAllowed    = `{"error":{"code":"unknown_tool","message":"tool %s is not available"}}`
	formatUnknownMode       = "unknown mode %q: use %s or %s"
	messageStepLimitReached = "Досягнуто ліміт кроків агента"
	wrapModelStep           = "model step: %w"
)
