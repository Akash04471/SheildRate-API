package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"api-rate-limiter/analytics"
	"api-rate-limiter/middleware"
	ratelimiter "api-rate-limiter/rate-limiter"
	"api-rate-limiter/stringutil"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("my_secret_key_change_in_production") // JWT Secret

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

	// Register multiple clients
	clients := []string{"client1", "client2", "client3"}
	for _, c := range clients {
		err := rl.RegisterClient(c, "mypassword")
		if err != nil {
			fmt.Println("Registration error:", err)
		}
	}

	// [ASYNC WORKFLOW] Setup asynchronous logger
	logChan := make(chan string, 100)
	go asyncLogger(logChan)

	// [ANALYTICS] Initialize the statistical analytics collector (5-minute window)
	collector := analytics.NewCollector(5 * time.Minute)
	defer collector.Shutdown()

	// API handler (Protected endpoint)
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t0 := time.Now()
		w.Header().Set("Content-Type", "application/json")

		// Extract claims to show personalized response
		var clientID string
		tokenStr := extractBearerToken(r.Header.Get("Authorization"))
		claims := &jwt.RegisteredClaims{}
		if tokenStr != "" {
			_, _ = jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) { return jwtKey, nil })
			clientID = claims.Subject
		}

		response := struct {
			Message string `json:"message"`
			Status  string `json:"status"`
			User    string `json:"user"`
		}{
			Message: "API request successful",
			Status:  "ok",
			User:    clientID,
		}
		json.NewEncoder(w).Encode(response)

		// [ANALYTICS] Record latency for statistical analysis
		latencyMs := float64(time.Since(t0).Microseconds()) / 1000.0
		collector.Record(clientID, latencyMs)

		// [STRING UTIL] Use structured log formatting and masked token
		maskedToken := stringutil.MaskToken(tokenStr, 8)
		logChan <- stringutil.FormatLogEntry(clientID, "success", fmt.Sprintf("API ok | latency=%.2fms | token=%s", latencyMs, maskedToken))
	})

	// Analytics handler — returns real-time statistical snapshot (JWT-protected)
	analyticsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(collector.Snapshot())
	})

	protectedAnalytics := middleware.CORSMiddleware(jwtMiddleware(analyticsHandler))

	// Wrap apiHandler with JWT and Rate Limiter Middlewares
	protectedAPIHandler := middleware.CORSMiddleware(
		jwtMiddleware(middleware.RateLimitMiddleware(rl, apiHandler)),
	)

	// Authentication handler
	authHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// [STRING UTIL] Sanitize and validate client ID
		authRequest.ClientID = stringutil.SanitizeInput(authRequest.ClientID)
		if errMsg, valid := stringutil.ValidateClientID(authRequest.ClientID); !valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
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

		fmt.Println("Authentication successful for client:", authRequest.ClientID)

		tokenString, err := generateJWT(authRequest.ClientID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate token"})
			return
		}

		tokenResponse := struct {
			Token     string `json:"token"`
			ClientID  string `json:"clientID"`
			Message   string `json:"message"`
			ExpiresAt time.Time `json:"expiresAt"`
		}{
			Token:     tokenString,
			ClientID:  authRequest.ClientID,
			Message:   "Authentication successful",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(tokenResponse)
	})

	authFinalHandler := middleware.CORSMiddleware(authHandler)

	// [STRING UTIL DEMO] Handler to showcase all stringutil functions via API
	stringUtilHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Accept input from query param or JSON body
		var input string
		if r.Method == http.MethodGet {
			input = r.URL.Query().Get("input")
		} else {
			var body struct{ Input string `json:"input"` }
			json.NewDecoder(r.Body).Decode(&body)
			input = body.Input
		}
		if input == "" {
			input = "Hello_World ExampleCamelCase <script>alert('xss')</script>"
		}

		// Run all string utility functions and collect results
		validateMsg, isValid := stringutil.ValidateClientID(input)
		result := struct {
			OriginalInput string `json:"original_input"`
			ValidateID    struct {
				Valid   bool   `json:"valid"`
				Message string `json:"message"`
			} `json:"validate_client_id"`
			MaskToken    string `json:"mask_token"`
			FormatLog    string `json:"format_log_entry"`
			Sanitized    string `json:"sanitize_input"`
			Slugified    string `json:"slugify"`
			Reversed     string `json:"reverse_string"`
			CamelToSnake string `json:"camel_to_snake"`
		}{}

		result.OriginalInput = input
		result.ValidateID.Valid = isValid
		result.ValidateID.Message = validateMsg
		if isValid {
			result.ValidateID.Message = "Valid client ID"
		}
		result.MaskToken = stringutil.MaskToken(input, 6)
		result.FormatLog = stringutil.FormatLogEntry("demo_client", "info", input)
		result.Sanitized = stringutil.SanitizeInput(input)
		result.Slugified = stringutil.Slugify(input)
		result.Reversed = stringutil.ReverseString(input)
		result.CamelToSnake = stringutil.CamelToSnake(input)

		json.NewEncoder(w).Encode(result)
	})

	protectedStringUtil := middleware.CORSMiddleware(jwtMiddleware(stringUtilHandler))

	// Static file server for the frontend
	fs := http.FileServer(http.Dir("./frontend"))

	// Register routes
	http.Handle("/api/status", protectedAPIHandler)
	http.Handle("/api/analytics", protectedAnalytics)
	http.Handle("/api/stringutil", protectedStringUtil) // [NEW] String utilities demo
	http.Handle("/auth/login", authFinalHandler)
	http.Handle("/", fs) // Serve frontend at root

	fmt.Println("Server running at http://localhost:8081")
	fmt.Println("Rate limiter service with JWT authentication and Async Logger enabled")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Println("Server failed:", err)
	}
}

// asyncLogger runs in an isolated goroutine to handle logging asynchronously.
// It now receives pre-formatted structured log entries from stringutil.FormatLogEntry.
func asyncLogger(logChan <-chan string) {
	for msg := range logChan {
		time.Sleep(50 * time.Millisecond)
		fmt.Printf("[ASYNC LOG] %s\n", msg)
	}
}

// generateJWT creates a new token with signed claims
func generateJWT(clientID string) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &jwt.RegisteredClaims{
		Subject:   clientID,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// jwtMiddleware validates the provided JWT token before allowing access
func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ignore OPTIONS for CORS preflight
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		tokenString := extractBearerToken(authHeader)

		if tokenString == "" {
			http.Error(w, "Missing authorization token", http.StatusUnauthorized)
			return
		}

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractBearerToken extracts the token from the "Bearer <token>" string
func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}
	return ""
}
