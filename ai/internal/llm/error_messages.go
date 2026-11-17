package llm

const (
	wrapCreateGeminiClient = "create Gemini client: %w"
	wrapEncodeToolCall     = "encode arguments of tool call %s: %w"
	wrapGenerateContent    = "generate content with %s: %w"
	wrapWaitBeforeCall     = "wait before model call: %w"
	formatBlockedResponse  = "%w: %s"
	formatWrapSentinel     = "%w: %w"
)
