package algorithms

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"shieldrate-api/rate-limiter/types"
)

func TestAlgorithms_Concurrency(t *testing.T) {
	const goroutines = 100
	const key = "concurrent-user-ip"

	tests := []struct {
		name          string
		limit         int
		createLimiter func(limit int) types.Limiter
	}{
		{
			name:  "FixedWindow",
			limit: 10,
			createLimiter: func(limit int) types.Limiter {
				return NewFixedWindow(limit, 1*time.Minute)
			},
		},
		{
			name:  "TokenBucket",
			limit: 10,
			createLimiter: func(limit int) types.Limiter {
				// Capacity = 10, negligible refill rate during fast burst
				return NewTokenBucket(10.0, 0.001)
			},
		},
		{
			name:  "SlidingWindowCounter",
			limit: 10,
			createLimiter: func(limit int) types.Limiter {
				return NewSlidingWindowCounter(limit, 1*time.Minute)
			},
		},
		{
			name:  "SlidingWindowLog",
			limit: 10,
			createLimiter: func(limit int) types.Limiter {
				return NewSlidingWindowLog(limit, 1*time.Minute)
			},
		},
		{
			name:  "LeakyBucket",
			limit: 10,
			createLimiter: func(limit int) types.Limiter {
				// Capacity = 10, negligible leak rate during fast burst
				return NewLeakyBucket(10, 0.001)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := tt.createLimiter(tt.limit)

			var wg sync.WaitGroup
			var allowedCount int64

			wg.Add(goroutines)
			for i := 0; i < goroutines; i++ {
				go func() {
					defer wg.Done()
					dec := limiter.Allow(key)
					if dec.Allowed {
						atomic.AddInt64(&allowedCount, 1)
					}
				}()
			}

			wg.Wait()

			totalAllowed := int(atomic.LoadInt64(&allowedCount))
			if totalAllowed > tt.limit {
				t.Errorf("%s concurrency test failed: allowedCount = %d, strictly expected <= %d",
					tt.name, totalAllowed, tt.limit)
			}
			if totalAllowed == 0 {
				t.Errorf("%s concurrency test failed: allowedCount = 0, expected > 0", tt.name)
			}
		})
	}
}
