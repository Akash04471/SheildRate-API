package algorithms

import (
	"math"
	"sync"
	"time"

	"shieldrate-api/rate-limiter/types"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type TokenBucket struct {
	mu         sync.Mutex
	buckets    map[string]*bucket
	capacity   float64
	refillRate float64
}

func NewTokenBucket(capacity, refillRate float64) *TokenBucket {
	return &TokenBucket{
		buckets:    make(map[string]*bucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

func (tb *TokenBucket) Allow(key string) types.Decision {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()

	b, exists := tb.buckets[key]
	if !exists {
		b = &bucket{
			tokens:     tb.capacity,
			lastRefill: now,
		}
		tb.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastRefill).Seconds()
		tokensToAdd := elapsed * tb.refillRate
		b.tokens = math.Min(tb.capacity, b.tokens+tokensToAdd)
		b.lastRefill = now
	}

	if b.tokens < 1.0 {
		neededTokens := 1.0 - b.tokens
		waitSeconds := neededTokens / tb.refillRate
		retryAfter := time.Duration(waitSeconds * float64(time.Second))

		return types.Decision{
			Allowed:    false,
			Remaining:  int(b.tokens),
			Limit:      int(tb.capacity),
			ResetAt:    now.Add(retryAfter),
			RetryAfter: retryAfter,
		}
	}

	b.tokens -= 1.0
	remaining := int(b.tokens)

	timeToFullSec := (tb.capacity - b.tokens) / tb.refillRate
	resetAt := now.Add(time.Duration(timeToFullSec * float64(time.Second)))

	return types.Decision{
		Allowed:    true,
		Remaining:  remaining,
		Limit:      int(tb.capacity),
		ResetAt:    resetAt,
		RetryAfter: 0,
	}
}
