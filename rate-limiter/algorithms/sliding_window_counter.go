package algorithms

import (
	"math"
	"sync"
	"time"

	"shieldrate-api/rate-limiter/types"
)

type windowState struct {
	prevCount   int
	currCount   int
	windowStart time.Time
}

type SlidingWindowCounter struct {
	mu      sync.Mutex
	windows map[string]*windowState
	limit   int
	window  time.Duration
}

func NewSlidingWindowCounter(limit int, window time.Duration) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		windows: make(map[string]*windowState),
		limit:   limit,
		window:  window,
	}
}

func (sw *SlidingWindowCounter) Allow(key string) types.Decision {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()

	ws, exists := sw.windows[key]
	if !exists {
		ws = &windowState{
			windowStart: now,
			prevCount:   0,
			currCount:   0,
		}
		sw.windows[key] = ws
	}

	elapsed := now.Sub(ws.windowStart)

	if elapsed >= 2*sw.window {
		ws.windowStart = now
		ws.prevCount = 0
		ws.currCount = 0
		elapsed = 0
	} else if elapsed >= sw.window {
		ws.windowStart = ws.windowStart.Add(sw.window)
		ws.prevCount = ws.currCount
		ws.currCount = 0
		elapsed = now.Sub(ws.windowStart)
	}

	elapsedFraction := elapsed.Seconds() / sw.window.Seconds()
	if elapsedFraction > 1.0 {
		elapsedFraction = 1.0
	}

	estimatedCount := float64(ws.prevCount)*(1.0-elapsedFraction) + float64(ws.currCount)

	windowResetAt := ws.windowStart.Add(sw.window)
	retryAfter := windowResetAt.Sub(now)
	if retryAfter < 0 {
		retryAfter = 0
	}

	if estimatedCount >= float64(sw.limit) {
		return types.Decision{
			Allowed:    false,
			Remaining:  0,
			Limit:      sw.limit,
			ResetAt:    windowResetAt,
			RetryAfter: retryAfter,
		}
	}

	ws.currCount++
	remaining := sw.limit - int(math.Floor(estimatedCount)) - 1
	if remaining < 0 {
		remaining = 0
	}

	return types.Decision{
		Allowed:    true,
		Remaining:  remaining,
		Limit:      sw.limit,
		ResetAt:    windowResetAt,
		RetryAfter: 0,
	}
}
