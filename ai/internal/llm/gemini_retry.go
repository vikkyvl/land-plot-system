package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

const (
	maximumRetries       = 3
	firstBackoff         = 2 * time.Second
	maximumRetryDelay    = time.Minute
	statusExhausted      = "RESOURCE_EXHAUSTED"
	retryInfoType        = "type.googleapis.com/google.rpc.RetryInfo"
	quotaFailureType     = "type.googleapis.com/google.rpc.QuotaFailure"
	dailyQuotaMarker     = "perday"
	detailTypeKey        = "@type"
	detailRetryDelayKey  = "retryDelay"
	detailViolationsKey  = "violations"
	detailQuotaIDKey     = "quotaId"
	detailQuotaMetricKey = "quotaMetric"
)

type sleepFunc func(sleepContext context.Context, delay time.Duration) error

type retryPolicy struct {
	sleep sleepFunc
}

func defaultRetryPolicy(sleep sleepFunc) retryPolicy {
	return retryPolicy{sleep: sleep}
}

func (policy retryPolicy) decide(err error, attempt int) (time.Duration, bool, error) {
	apiError, ok := errors.AsType[genai.APIError](err)
	if !ok {
		return 0, false, err
	}

	rateLimited := apiError.Code == http.StatusTooManyRequests || apiError.Status == statusExhausted
	unavailable := apiError.Code == http.StatusServiceUnavailable || apiError.Code == http.StatusInternalServerError

	switch {
	case rateLimited && isDailyQuotaError(apiError):
		return 0, false, fmt.Errorf(formatWrapSentinel, ErrDailyQuotaExhausted, err)
	case !rateLimited && !unavailable:
		return 0, false, err
	case attempt > maximumRetries:
		return 0, false, exhaustedRetries(rateLimited, err)
	}

	delay := firstBackoff << (attempt - 1)
	if suggested, found := suggestedRetryDelay(apiError); found {
		delay = suggested
	}

	if delay > maximumRetryDelay {
		return 0, false, exhaustedRetries(rateLimited, err)
	}

	return delay, true, nil
}

func exhaustedRetries(rateLimited bool, err error) error {
	if rateLimited {
		return fmt.Errorf(formatWrapSentinel, ErrRateLimited, err)
	}

	return err
}

func isDailyQuota(err error) bool {
	return errors.Is(err, ErrDailyQuotaExhausted)
}

func isDailyQuotaError(apiError genai.APIError) bool {
	for _, detail := range apiError.Details {
		if detail[detailTypeKey] != quotaFailureType {
			continue
		}

		violations, _ := detail[detailViolationsKey].([]any)
		for _, violation := range violations {
			fields, _ := violation.(map[string]any)
			quotaID, _ := fields[detailQuotaIDKey].(string)
			quotaMetric, _ := fields[detailQuotaMetricKey].(string)

			if strings.Contains(strings.ToLower(quotaID+quotaMetric), dailyQuotaMarker) {
				return true
			}
		}
	}

	return false
}

func suggestedRetryDelay(apiError genai.APIError) (time.Duration, bool) {
	for _, detail := range apiError.Details {
		if detail[detailTypeKey] != retryInfoType {
			continue
		}

		value, _ := detail[detailRetryDelayKey].(string)

		delay, err := time.ParseDuration(value)
		if err == nil && delay > 0 {
			return delay, true
		}
	}

	return 0, false
}

type throttle struct {
	mutex    sync.Mutex
	interval time.Duration
	next     time.Time
	now      func() time.Time
	sleep    sleepFunc
}

func newThrottle(interval time.Duration, now func() time.Time, sleep sleepFunc) *throttle {
	return &throttle{interval: interval, now: now, sleep: sleep}
}

func (limiter *throttle) wait(waitContext context.Context) error {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	if delay := limiter.next.Sub(limiter.now()); delay > 0 {
		if err := limiter.sleep(waitContext, delay); err != nil {
			return err
		}
	}

	limiter.next = limiter.now().Add(limiter.interval)

	return nil
}

func sleepContext(sleepContext context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-sleepContext.Done():
		return sleepContext.Err()
	case <-timer.C:
		return nil
	}
}
