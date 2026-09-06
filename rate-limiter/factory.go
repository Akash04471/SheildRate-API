package ratelimiter

import (
	"time"

	"shieldrate-api/rate-limiter/algorithms"
)

// AlgorithmType defines supported rate limiting algorithms.
type AlgorithmType string

const (
	FixedWindow          AlgorithmType = "fixed_window"
	SlidingWindowLog     AlgorithmType = "sliding_window_log"
	SlidingWindowCounter AlgorithmType = "sliding_window_counter"
	TokenBucket          AlgorithmType = "token_bucket"
	LeakyBucket          AlgorithmType = "leaky_bucket"
)

// NewLimiter acts as a Factory function creating and returning the specified concrete Limiter implementation.
// Defaults to FixedWindow if an unrecognized algorithm type is passed.
func NewLimiter(algo AlgorithmType, limit int, window time.Duration) Limiter {
	switch algo {
	case FixedWindow:
		return algorithms.NewFixedWindow(limit, window)
	case SlidingWindowLog:
		return algorithms.NewSlidingWindowLog(limit, window)
	case SlidingWindowCounter:
		return algorithms.NewSlidingWindowCounter(limit, window)
	case TokenBucket:
		// Calculate refill rate as tokens/sec
		refillRate := float64(limit) / window.Seconds()
		return algorithms.NewTokenBucket(float64(limit), refillRate)
	case LeakyBucket:
		// Calculate leak rate as requests processed/sec
		leakRate := float64(limit) / window.Seconds()
		return algorithms.NewLeakyBucket(limit, leakRate)
	default:
		return algorithms.NewFixedWindow(limit, window)
	}
}
