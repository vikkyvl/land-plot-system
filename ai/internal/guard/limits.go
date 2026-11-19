package guard

type TokenBudget struct {
	limit int
	spent int
}

func NewTokenBudget(limit int) *TokenBudget {
	return &TokenBudget{limit: limit}
}

func (budget *TokenBudget) Spend(tokens int) {
	budget.spent += max(tokens, 0)
}

func (budget *TokenBudget) Spent() int {
	return budget.spent
}

func (budget *TokenBudget) Remaining() int {
	return max(budget.limit-budget.spent, 0)
}

func (budget *TokenBudget) Check() error {
	if budget.spent >= budget.limit {
		return ErrTokenBudgetExhausted
	}

	return nil
}

type StepLimiter struct {
	limit int
	taken int
}

func NewStepLimiter(limit int) *StepLimiter {
	return &StepLimiter{limit: limit}
}

func (limiter *StepLimiter) Take() error {
	if limiter.taken >= limiter.limit {
		return ErrStepLimitReached
	}

	limiter.taken++

	return nil
}

func (limiter *StepLimiter) Taken() int {
	return limiter.taken
}
