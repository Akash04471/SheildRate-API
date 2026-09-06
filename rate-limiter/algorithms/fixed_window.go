package algorithms

import (
	"sync"
	"time"

	"shieldrate-api/rate-limiter/types"
)

type clientState struct {
	requestCount int
	resetTime    time.Time
}

// FixedWindow implements the Limiter interface using a fixed window algorithm.
type FixedWindow struct {
	mu      sync.Mutex
	clients map[string]*clientState
	limit   int
	window  time.Duration
}

// NewFixedWindow initializes and returns a new FixedWindow rate limiter.
func NewFixedWindow(limit int, window time.Duration) *FixedWindow {
	return &FixedWindow{
		clients: make(map[string]*clientState),
		limit:   limit,
		window:  window,
	}
}

// Allow evaluates whether a request for a given key is permitted under the fixed window strategy.
func (fw *FixedWindow) Allow(key string) types.Decision {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	client, exists := fw.clients[key]
	if !exists {
		client = &clientState{}
		fw.clients[key] = client
	}

	now := time.Now()

	// Reset window if expired or not set
	if client.resetTime.IsZero() || now.After(client.resetTime) {
		client.requestCount = 0
		client.resetTime = now.Add(fw.window)
	}

	retryAfter := client.resetTime.Sub(now)
	if retryAfter < 0 {
		retryAfter = 0
	}

	if client.requestCount >= fw.limit {
		return types.Decision{
			Allowed:    false,
			Remaining:  0,
			Limit:      fw.limit,
			ResetAt:    client.resetTime,
			RetryAfter: retryAfter,
		}
	}

	client.requestCount++
	remaining := fw.limit - client.requestCount

	return types.Decision{
		Allowed:    true,
		Remaining:  remaining,
		Limit:      fw.limit,
		ResetAt:    client.resetTime,
		RetryAfter: retryAfter,
	}
}
