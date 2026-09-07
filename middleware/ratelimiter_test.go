package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"shieldrate-api/rate-limiter/algorithms"
)

type blockedResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	RetryAfter string `json:"retry_after"`
}

func TestRateLimiterMiddleware(t *testing.T) {
	limit := 2
	window := 1 * time.Minute
	limiter := algorithms.NewFixedWindow(limit, window)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handlerToTest := RateLimiter(limiter)(dummyHandler)

	// 1 & 2. Fire requests up to the limit and assert HTTP 200 OK responses with headers
	for i := 1; i <= limit; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()

		handlerToTest.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status HTTP 200 OK, got %d", i, rec.Code)
		}

		wantRemaining := limit - i
		if gotLimit := rec.Header().Get("X-RateLimit-Limit"); gotLimit != strconv.Itoa(limit) {
			t.Errorf("request %d: header X-RateLimit-Limit = %q, want %q", i, gotLimit, strconv.Itoa(limit))
		}
		if gotRemaining := rec.Header().Get("X-RateLimit-Remaining"); gotRemaining != strconv.Itoa(wantRemaining) {
			t.Errorf("request %d: header X-RateLimit-Remaining = %q, want %q", i, gotRemaining, strconv.Itoa(wantRemaining))
		}
	}

	// 3 & 4. Fire one request beyond the limit and assert HTTP 429, JSON response, and Retry-After headers
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	handlerToTest.ServeHTTP(rec, req)

	// Assert HTTP Status 429
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status HTTP 429 Too Many Requests, got %d", rec.Code)
	}

	// Assert Headers
	if gotLimit := rec.Header().Get("X-RateLimit-Limit"); gotLimit != strconv.Itoa(limit) {
		t.Errorf("blocked request: header X-RateLimit-Limit = %q, want %q", gotLimit, strconv.Itoa(limit))
	}
	if gotRemaining := rec.Header().Get("X-RateLimit-Remaining"); gotRemaining != "0" {
		t.Errorf("blocked request: header X-RateLimit-Remaining = %q, want %q", gotRemaining, "0")
	}

	retryAfterHeader := rec.Header().Get("Retry-After")
	if retryAfterHeader == "" {
		t.Errorf("blocked request: expected Retry-After header to be present")
	} else {
		retrySec, err := strconv.Atoi(retryAfterHeader)
		if err != nil || retrySec <= 0 {
			t.Errorf("blocked request: Retry-After header %q is invalid or <= 0", retryAfterHeader)
		}
	}

	// Parse JSON body and assert fields
	var resp blockedResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response body: %v", err)
	}

	if resp.Status != "blocked" {
		t.Errorf("JSON status = %q, want %q", resp.Status, "blocked")
	}
	if resp.Message != "Rate limit exceeded" {
		t.Errorf("JSON message = %q, want %q", resp.Message, "Rate limit exceeded")
	}
	if resp.RetryAfter == "" {
		t.Errorf("JSON retry_after field is empty")
	}
}
