package types

import (
	"time"
)

// Decision represents the outcome of a rate-limiting evaluation for a request.
type Decision struct {
	Allowed    bool          // Indicates whether the request is allowed to proceed.
	Remaining  int           // The number of remaining requests allowed within the current time window.
	Limit      int           // The maximum request limit configured for the window.
	ResetAt    time.Time     // The time when the rate limit counter resets.
	RetryAfter time.Duration // The duration a client must wait before retrying if rate-limited.
}

// Limiter defines the core interface for rate-limiting strategies.
type Limiter interface {
	Allow(key string) Decision
}

// Store defines the storage backend interface for rate limiter data state management.
type Store interface {
	Increment(key string, window time.Duration) (count int, resetAt time.Time, err error)
	Get(key string) (count int, resetAt time.Time, err error)
}
