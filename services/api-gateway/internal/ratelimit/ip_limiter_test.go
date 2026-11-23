package ratelimit

import (
	"testing"
	"time"
)

type fakeClock struct{ current time.Time }

func (clock *fakeClock) now() time.Time { return clock.current }

func (clock *fakeClock) advance(duration time.Duration) { clock.current = clock.current.Add(duration) }

func newTestLimiter(perMinute int) (*IPLimiter, *fakeClock) {
	clock := &fakeClock{current: time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)}

	return newIPLimiter(perMinute, clock.now), clock
}

func TestLimiterAllowsTheBurstAndThenRejects(t *testing.T) {
	limiter, _ := newTestLimiter(60)

	for request := range 60 {
		if allowed, _ := limiter.Allow("192.0.2.1"); !allowed {
			t.Fatalf("request %d was rejected", request+1)
		}
	}

	allowed, retryAfter := limiter.Allow("192.0.2.1")
	if allowed || retryAfter <= 0 || retryAfter > time.Second {
		t.Fatalf("61st request: allowed = %v, retry after = %v", allowed, retryAfter)
	}
}

func TestLimiterCountsEveryAddressSeparately(t *testing.T) {
	limiter, _ := newTestLimiter(1)

	first, _ := limiter.Allow("192.0.2.1")
	second, _ := limiter.Allow("192.0.2.2")
	repeated, _ := limiter.Allow("192.0.2.1")

	if !first || !second || repeated {
		t.Fatalf("first = %v, second = %v, repeated = %v", first, second, repeated)
	}
}

func TestLimiterRefillsOverTime(t *testing.T) {
	limiter, clock := newTestLimiter(60)
	for range 60 {
		limiter.Allow("192.0.2.1")
	}

	clock.advance(time.Second)

	if allowed, _ := limiter.Allow("192.0.2.1"); !allowed {
		t.Fatal("request after one second was rejected")
	}
}

func TestRejectedRequestsDoNotConsumeTokens(t *testing.T) {
	limiter, clock := newTestLimiter(60)
	for range 70 {
		limiter.Allow("192.0.2.1")
	}

	clock.advance(time.Second)

	if allowed, _ := limiter.Allow("192.0.2.1"); !allowed {
		t.Fatal("rejected requests consumed future tokens")
	}
}

func TestLimiterForgetsIdleAddresses(t *testing.T) {
	limiter, clock := newTestLimiter(60)
	limiter.Allow("192.0.2.1")

	clock.advance(idleVisitorTTL)
	limiter.Allow("192.0.2.2")

	if _, kept := limiter.visitors["192.0.2.1"]; kept || len(limiter.visitors) != 1 {
		t.Fatalf("visitors = %v", limiter.visitors)
	}
}
