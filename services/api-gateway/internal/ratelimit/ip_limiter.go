package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const idleVisitorTTL = time.Minute

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPLimiter struct {
	mutex     sync.Mutex
	visitors  map[string]*visitor
	limit     rate.Limit
	burst     int
	lastSweep time.Time
	now       func() time.Time
}

func NewIPLimiter(perMinute int) *IPLimiter {
	return newIPLimiter(perMinute, time.Now)
}

func newIPLimiter(perMinute int, now func() time.Time) *IPLimiter {
	return &IPLimiter{
		visitors:  make(map[string]*visitor),
		limit:     rate.Limit(float64(perMinute) / time.Minute.Seconds()),
		burst:     perMinute,
		lastSweep: now(),
		now:       now,
	}
}

func (ipLimiter *IPLimiter) Allow(key string) (bool, time.Duration) {
	ipLimiter.mutex.Lock()
	defer ipLimiter.mutex.Unlock()

	now := ipLimiter.now()
	ipLimiter.sweep(now)

	current, exists := ipLimiter.visitors[key]
	if !exists {
		current = &visitor{limiter: rate.NewLimiter(ipLimiter.limit, ipLimiter.burst)}
		ipLimiter.visitors[key] = current
	}

	current.lastSeen = now

	reservation := current.limiter.ReserveN(now, 1)
	if delay := reservation.DelayFrom(now); delay > 0 {
		reservation.CancelAt(now)

		return false, delay
	}

	return true, 0
}

func (ipLimiter *IPLimiter) sweep(now time.Time) {
	if now.Sub(ipLimiter.lastSweep) < idleVisitorTTL {
		return
	}

	ipLimiter.lastSweep = now

	for key, idle := range ipLimiter.visitors {
		if now.Sub(idle.lastSeen) >= idleVisitorTTL {
			delete(ipLimiter.visitors, key)
		}
	}
}
