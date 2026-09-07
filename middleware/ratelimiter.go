package middleware

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	ratelimiter "shieldrate-api/rate-limiter"
)

func RateLimiter(limiter ratelimiter.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Extracting client identifier
			clientID := ratelimiter.GetClientID(r.RemoteAddr)
			fmt.Println("Incoming request from:", clientID)

			// Rate limit check
			decision := limiter.Allow(clientID)

			// Set rate limit headers for all responses
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(decision.Remaining))

			// Blocked request
			if !decision.Allowed {
				fmt.Println("Request blocked:", clientID)

				retryAfterSec := int(math.Ceil(decision.RetryAfter.Seconds()))
				if retryAfterSec < 1 && decision.RetryAfter > 0 {
					retryAfterSec = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfterSec))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)

				// Anonymous struct
				response := struct {
					Status     string `json:"status"`
					Message    string `json:"message"`
					RetryAfter string `json:"retry_after"`
				}{
					Status:     "blocked",
					Message:    "Rate limit exceeded",
					RetryAfter: time.Now().Add(decision.RetryAfter).Format("15:04:05"),
				}

				json.NewEncoder(w).Encode(response)
				return
			}

			// Allowed request
			fmt.Println("Request allowed:", clientID, "| Remaining:", decision.Remaining)
			next.ServeHTTP(w, r)
		})
	}
}

