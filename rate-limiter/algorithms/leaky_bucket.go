package algorithms

import (
	"math"
	"sync"
	"time"

	"shieldrate-api/rate-limiter/types"
)

type leakyState struct {
	water    float64
	lastLeak time.Time
}

type LeakyBucket struct {
	mu       sync.Mutex
	buckets  map[string]*leakyState
	capacity int
	leakRate float64
}

func NewLeakyBucket(capacity int, leakRate float64) *LeakyBucket {
	return &LeakyBucket{
		buckets:  make(map[string]*leakyState),
		capacity: capacity,
		leakRate: leakRate,
	}
}

func (lb *LeakyBucket) Allow(key string) types.Decision {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	now := time.Now()

	b, exists := lb.buckets[key]
	if !exists {
		b = &leakyState{
			water:    0,
			lastLeak: now,
		}
		lb.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastLeak).Seconds()
		leaked := elapsed * lb.leakRate
		b.water = math.Max(0, b.water-leaked)
		b.lastLeak = now
	}

	if b.water+1.0 > float64(lb.capacity) {
		excessWater := (b.water + 1.0) - float64(lb.capacity)
		timeToSlotSec := excessWater / lb.leakRate
		retryAfter := time.Duration(timeToSlotSec * float64(time.Second))

		timeToEmptySec := b.water / lb.leakRate
		resetAt := now.Add(time.Duration(timeToEmptySec * float64(time.Second)))

		return types.Decision{
			Allowed:    false,
			Remaining:  0,
			Limit:      lb.capacity,
			ResetAt:    resetAt,
			RetryAfter: retryAfter,
		}
	}

	b.water += 1.0
	remaining := lb.capacity - int(math.Ceil(b.water))
	if remaining < 0 {
		remaining = 0
	}

	timeToEmptySec := b.water / lb.leakRate
	resetAt := now.Add(time.Duration(timeToEmptySec * float64(time.Second)))

	return types.Decision{
		Allowed:    true,
		Remaining:  remaining,
		Limit:      lb.capacity,
		ResetAt:    resetAt,
		RetryAfter: 0,
	}
}
