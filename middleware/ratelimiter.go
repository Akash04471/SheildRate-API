package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
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

			// Blocked request
			if !decision.Allowed {
				fmt.Println("Request blocked:", clientID)

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

