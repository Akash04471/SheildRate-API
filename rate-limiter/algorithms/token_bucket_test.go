package algorithms

import (
	"testing"
	"time"
)

// Note: Timing-based assertions use tolerance/threshold checks
// (e.g., duration >= expected - tolerance && duration <= expected + tolerance)
// rather than exact equality to avoid flaky tests due to OS scheduling and non-deterministic execution times.

func TestTokenBucket_Allow(t *testing.T) {
	type step struct {
		key                 string
		sleepBefore         time.Duration
		wantAllowed         bool
		wantRemaining       int
		wantLimit           int
		expectedRetryAfter  time.Duration
		retryAfterTolerance time.Duration
	}

	tests := []struct {
		name       string
		capacity   float64
		refillRate float64
		steps      []step
	}{
		{
			name:       "1. Bucket starts full at capacity and allows requests up to capacity",
			capacity:   3.0,
			refillRate: 1.0, // 1 token / sec
			steps: []step{
				{key: "client-1", wantAllowed: true, wantRemaining: 2, wantLimit: 3},
				{key: "client-1", wantAllowed: true, wantRemaining: 1, wantLimit: 3},
				{key: "client-1", wantAllowed: true, wantRemaining: 0, wantLimit: 3},
			},
		},
		{
			name:       "2. Request beyond capacity with no elapsed time is denied",
			capacity:   2.0,
			refillRate: 10.0,
			steps: []step{
				{key: "client-1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "client-1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{key: "client-1", wantAllowed: false, wantRemaining: 0, wantLimit: 2},
			},
		},
		{
			name:       "3. After sleeping long enough for at least 1 token to refill, next request is allowed",
			capacity:   2.0,
			refillRate: 20.0, // 1 token per 50ms
			steps: []step{
				{key: "client-1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "client-1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{key: "client-1", wantAllowed: false, wantRemaining: 0, wantLimit: 2},
				{
					key:           "client-1",
					sleepBefore:   60 * time.Millisecond, // > 50ms, refills at least 1 token
					wantAllowed:   true,
					wantRemaining: 0,
					wantLimit:     2,
				},
			},
		},
		{
			name:       "4. RetryAfter value returned on denial is proportional to missing tokens divided by refillRate",
			capacity:   2.0,
			refillRate: 20.0, // 1 token per 50ms (0.05s)
			steps: []step{
				{key: "client-1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "client-1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{
					key:                 "client-1",
					wantAllowed:         false,
					wantRemaining:       0,
					wantLimit:           2,
					expectedRetryAfter:  50 * time.Millisecond,
					retryAfterTolerance: 20 * time.Millisecond,
				},
			},
		},
		{
			name:       "5. Independent keys have independent bucket state",
			capacity:   2.0,
			refillRate: 1.0,
			steps: []step{
				{key: "client-A", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "client-A", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{key: "client-A", wantAllowed: false, wantRemaining: 0, wantLimit: 2},
				// client-B should start fresh at full capacity
				{key: "client-B", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "client-B", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTokenBucket(tt.capacity, tt.refillRate)

			for i, st := range tt.steps {
				if st.sleepBefore > 0 {
					time.Sleep(st.sleepBefore)
				}

				got := tb.Allow(st.key)

				if got.Allowed != st.wantAllowed {
					t.Errorf("step %d (key %q): Allowed = %v, want %v", i, st.key, got.Allowed, st.wantAllowed)
				}
				if got.Remaining != st.wantRemaining {
					t.Errorf("step %d (key %q): Remaining = %d, want %d", i, st.key, got.Remaining, st.wantRemaining)
				}
				if got.Limit != st.wantLimit {
					t.Errorf("step %d (key %q): Limit = %d, want %d", i, st.key, got.Limit, st.wantLimit)
				}

				if st.expectedRetryAfter > 0 {
					diff := got.RetryAfter - st.expectedRetryAfter
					if diff < 0 {
						diff = -diff
					}
					if diff > st.retryAfterTolerance {
						t.Errorf("step %d (key %q): RetryAfter = %v, want %v (± %v tolerance)",
							i, st.key, got.RetryAfter, st.expectedRetryAfter, st.retryAfterTolerance)
					}
				}
			}
		})
	}
}
