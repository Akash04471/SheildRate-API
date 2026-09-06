package config

import (
	"os"
	"time"
)

// Configuration-level variables
// Using var so they are accessible across packages

var (
	// Maximum number of requests allowed per client
	RequestLimit int = 10

	// Time window for rate limiting
	TimeWindow time.Duration = 1 * time.Minute

	// Server port number
	ServerPort string = ":8080"

	// Rate limiting algorithm strategy (fixed_window, sliding_window_log, sliding_window_counter, token_bucket, leaky_bucket)
	Algorithm string = "fixed_window"
)

func init() {
	if envAlgo := os.Getenv("RATE_LIMIT_ALGORITHM"); envAlgo != "" {
		Algorithm = envAlgo
	}
}
