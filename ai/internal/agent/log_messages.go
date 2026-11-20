package agent

const (
	logAgentModelStep     = "agent model step"
	logAgentStopped       = "agent stopped by a limit"
	logSecretRedacted     = "secret redacted from the model answer"
	logSuspiciousToolData = "suspicious phrase masked in tool result"
	logToolCall           = "tool call"
	logToolNotAllowed     = "model called a tool outside the allowlist"
	logUserInputRejected  = "user input rejected"
)
