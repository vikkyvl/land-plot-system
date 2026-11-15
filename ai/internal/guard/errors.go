package guard

import "errors"

var (
	ErrEmptyInput           = errors.New("запит порожній")
	ErrInputTooLong         = errors.New("запит задовгий")
	ErrSuspiciousInput      = errors.New("запит відхилено: він схожий на спробу prompt injection")
	ErrTokenBudgetExhausted = errors.New("бюджет токенів вичерпано")
	ErrStepLimitReached     = errors.New("досягнуто ліміт кроків агента")
)
