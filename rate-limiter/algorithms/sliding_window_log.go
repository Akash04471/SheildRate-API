package algorithms

import (
	"sync"
	"time"

	"shieldrate-api/rate-limiter/types"
)

type SlidingWindowLog struct {
	mu      sync.Mutex
	logs    map[string][]time.Time
	limit   int
	window  time.Duration
}

func NewSlidingWindowLog(limit int, window time.Duration) *SlidingWindowLog {
	return &SlidingWindowLog{
		logs:   make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
}

func (swl *SlidingWindowLog) Allow(key string) types.Decision {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-swl.window)

	timestamps := swl.logs[key]

	validIdx := 0
	for i, t := range timestamps {
		if t.After(cutoff) {
			validIdx = i
			break
		}
		if i == len(timestamps)-1 {
			validIdx = len(timestamps)
		}
	}

	if validIdx > 0 {
		if validIdx >= len(timestamps) {
			timestamps = nil
		} else {
			timestamps = timestamps[validIdx:]
		}
	}

	if len(timestamps) >= swl.limit {
		oldest := timestamps[0]
		resetAt := oldest.Add(swl.window)
		retryAfter := resetAt.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}

		swl.logs[key] = timestamps

		return types.Decision{
			Allowed:    false,
			Remaining:  0,
			Limit:      swl.limit,
			ResetAt:    resetAt,
			RetryAfter: retryAfter,
		}
	}

	timestamps = append(timestamps, now)
	swl.logs[key] = timestamps

	remaining := swl.limit - len(timestamps)
	oldest := timestamps[0]
	resetAt := oldest.Add(swl.window)

	return types.Decision{
		Allowed:    true,
		Remaining:  remaining,
		Limit:      swl.limit,
		ResetAt:    resetAt,
		RetryAfter: 0,
	}
}
