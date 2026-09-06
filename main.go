package main

import (
	"fmt"
	"net/http"

	"shieldrate-api/config"
	"shieldrate-api/middleware"
	ratelimiter "shieldrate-api/rate-limiter"
)

// CORS middleware for browser-based frontend
func enableCORS(next http.Handler) http.Handler { //cross-origin requests
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {

	fmt.Println("Starting API Rate Limiter Server...")
	fmt.Printf("Configured Algorithm: %s (Limit: %d, Window: %v)\n", config.Algorithm, config.RequestLimit, config.TimeWindow)

	// Initialize Rate Limiter using Factory
	limiter := ratelimiter.NewLimiter(
		ratelimiter.AlgorithmType(config.Algorithm),
		config.RequestLimit,
		config.TimeWindow,
	)

	// Base API handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Request successful")
	})

	// Route with middleware chain
	http.Handle(
		"/api/test",
		enableCORS(
			middleware.RateLimiter(limiter)(handler),
		),
	)

	fmt.Println("Server listening on port", config.ServerPort)

	err := http.ListenAndServe(config.ServerPort, nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}
