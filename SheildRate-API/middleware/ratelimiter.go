package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"api-rate-limiter/config"
	ratelimiter "api-rate-limiter/rate-limiter"
)

func RateLimitMiddleware(rl *ratelimiter.RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		clientID := r.Header.Get("X-Client-ID")

		// Check if client is authenticated
		if clientID == "" {
			clientID = r.RemoteAddr
		}

		// token validation is handled by jwtMiddleware


		allowed, err := rl.Allow(clientID, config.MaxRequests, config.WindowDuration)

		if err != nil {
			fmt.Println("Error:", err)
			respondWithError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Anonymous struct - JSON marshalling
		response := struct {
			ClientID string `json:"client_id"`
			Allowed  bool   `json:"allowed"`
			Message  string `json:"message"`
		}{}

		if !allowed {
			fmt.Println("Rate limit exceeded for:", clientID)
			response.ClientID = clientID
			response.Allowed = false
			response.Message = "Rate limit exceeded"
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(response)
			return
		}

		fmt.Println("Request allowed for:", clientID)
		next.ServeHTTP(w, r)
	})
}

// Helper function to respond with error (JSON marshalling)
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := struct {
		Error   string `json:"error"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	}{
		Error:   http.StatusText(statusCode),
		Status:  statusCode,
		Message: message,
	}

	json.NewEncoder(w).Encode(errorResponse)
}
