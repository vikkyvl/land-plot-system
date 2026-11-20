package llm

import "errors"

var (
	ErrMissingAPIKey       = errors.New("LLM_API_KEY не задано")
	ErrRateLimited         = errors.New("ліміт безкоштовного тарифу вичерпано, спробуйте пізніше")
	ErrDailyQuotaExhausted = errors.New("денний ліміт безкоштовного тарифу вичерпано")
	ErrOutputTruncated     = errors.New("відповідь моделі обрізано лімітом LLM_MAX_OUTPUT_TOKENS")
	ErrEmptyResponse       = errors.New("модель повернула порожню відповідь")
	ErrResponseBlocked     = errors.New("модель відмовилась відповідати")
	ErrNoScriptedResponse  = errors.New("fake provider has no more responses")
)
