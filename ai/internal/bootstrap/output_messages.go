package bootstrap

const (
	formatHeader        = "Режим: %s, модель: %s\nЗапит: %s\n\n"
	formatModelStep     = "[модель %d] токени: вхід %d, вихід %d, роздуми %d, усього %d\n"
	formatToolCall      = "  → %s %s\n"
	formatToolResult    = "  ← %s\n"
	formatToolError     = "  ← помилка: %s\n"
	formatAnswer        = "\nВідповідь:\n%s\n"
	formatStopReason    = "Причина зупинки: %v\n"
	formatFailure       = "\nПомилка: %v\n"
	formatSummary       = "\nКроків з інструментами: %d. Викликів моделі: %d. Токенів: %s.\n"
	formatUsageTotals   = "%d (вхід %d, вихід %d, роздуми %d)"
	formatUsage         = "usage: fc-client [-mode non-reasoning|reasoning] [-v] \"запит\"\n"
	formatMCPUsage      = "usage: mcp-server [-transport stdio|http]\n"
	formatA2ACallUsage  = "usage: a2a-call [-timeout 3m] <agent-url> \"текст задачі\"\n"
	formatAgentCard     = "Агент: %s %s, skills: %s\nЗадача: %s\n\n"
	formatTaskEvent     = "[%s] задача %s\n"
	formatStatusEvent   = "[%s] %s\n"
	formatArtifactEvent = "[артефакт %s] %s\n"
	formatMessageEvent  = "[повідомлення] %s\n"
	ellipsis            = "…"
)
