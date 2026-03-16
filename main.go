package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"api-rate-limiter/middleware"
	ratelimiter "api-rate-limiter/rate-limiter"
)

func main() {

	rl := ratelimiter.NewRateLimiter()
	defer rl.Shutdown()
	// Pointer demonstration
	value := 5

	ratelimiter.IncrementByValue(value)
	fmt.Println("After call by value:", value)

	ratelimiter.IncrementByPointer(&value)
	fmt.Println("After call by pointer:", value)

	// JSON demonstration
	client := ratelimiter.Client{RequestCount: 3}
	jsonData, _ := ratelimiter.ToJSON(client)
	fmt.Println("JSON:", string(jsonData))

	parsedClient, _ := ratelimiter.FromJSON(jsonData)
	fmt.Println("Parsed RequestCount:", parsedClient.RequestCount)

	// bcrypt demo - register test client
	err := rl.RegisterClient("client1", "mypassword")
	if err != nil {
		fmt.Println("Registration error:", err)
	}

	// Authenticate test client
	// Register multiple clients
	clients := []string{"client1", "client2", "client3"}

	for _, c := range clients {
		err := rl.RegisterClient(c, "mypassword")
		if err != nil {
			fmt.Println("Registration error:", err)
		}
	}

	// API handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		}{
			Message: "API request successful",
			Status:  "ok",
		}
		json.NewEncoder(w).Encode(response)
	})

	// Authentication handler
	authHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body for authentication (JSON unmarshalling)
		var authRequest struct {
			ClientID string `json:"clientID"`
			Password string `json:"password"`
		}

		err := json.NewDecoder(r.Body).Decode(&authRequest)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid request format",
			})
			return
		}

		// Authenticate using bcrypt
		err = rl.Authenticate(authRequest.ClientID, authRequest.Password)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid credentials",
			})
			return
		}

		// Generate token response (JSON marshalling)
		tokenResponse := struct {
			Token     string `json:"token"`
			ClientID  string `json:"clientID"`
			Message   string `json:"message"`
			ExpiresAt string `json:"expiresAt"`
		}{
			Token:     generateSimpleToken(authRequest.ClientID),
			ClientID:  authRequest.ClientID,
			Message:   "Authentication successful",
			ExpiresAt: "3600s",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tokenResponse)
	})

	// Apply middleware
	finalHandler := middleware.CORSMiddleware(
		middleware.RateLimitMiddleware(rl, handler),
	)

	authFinalHandler := middleware.CORSMiddleware(authHandler)

	// Register routes
	http.Handle("/", finalHandler)
	http.Handle("/auth/login", authFinalHandler)

	fmt.Println("Server running at http://localhost:8080")
	fmt.Println("Rate limiter service with authentication enabled")
	http.ListenAndServe(":8080", nil)
}

// Simple token generation function
func generateSimpleToken(clientID string) string {
	// In production, use JWT or similar
	// For demo: return simple token
	return "demo_token_" + clientID
}
