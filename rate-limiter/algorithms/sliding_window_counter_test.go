package algorithms

import (
	"testing"
	"time"
)

func TestSlidingWindowCounter_Allow(t *testing.T) {
	type step struct {
		key             string
		sleepBefore     time.Duration
		wantAllowed     bool
		wantRemaining   int
		wantLimit       int
		checkRetryAfter bool
	}

	tests := []struct {
		name   string
		limit  int
		window time.Duration
		steps  []step
	}{
		{
			name:   "1. Allows requests up to limit",
			limit:  3,
			window: 100 * time.Millisecond,
			steps: []step{
				{key: "client-1", wantAllowed: true, wantRemaining: 2, wantLimit: 3},
				{key: "client-1", wantAllowed: true, wantRemaining: 1, wantLimit: 3},
				{key: "client-1", wantAllowed: true, wantRemaining: 0, wantLimit: 3},
			},
		},
		{
			name:   "2. Denies requests exceeding limit",
			limit:  2,
			window: 100 * time.Millisecond,
			steps: []step{
				{key: "client-1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "client-1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{
					key:             "client-1",
					wantAllowed:     false,
					wantRemaining:   0,
					wantLimit:       2,
					checkRetryAfter: true,
				},
			},
		},
		{
			name:   "3. Recovery after time passes (window shifts and counter resets)",
			limit:  2,
			window: 60 * time.Millisecond,
			steps: []step{
				{key: "client-1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "client-1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{key: "client-1", wantAllowed: false, wantRemaining: 0, wantLimit: 2, checkRetryAfter: true},
				{
					key:           "client-1",
					sleepBefore:   130 * time.Millisecond, // > 2 * window (full reset)
					wantAllowed:   true,
					wantRemaining: 1,
					wantLimit:     2,
				},
			},
		},
		{
			name:   "4. Independent state per key",
			limit:  2,
			window: 100 * time.Millisecond,
			steps: []step{
				{key: "ip-1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "ip-1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{key: "ip-1", wantAllowed: false, wantRemaining: 0, wantLimit: 2},
				{key: "ip-2", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "ip-2", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := NewSlidingWindowCounter(tt.limit, tt.window)

			for i, st := range tt.steps {
				if st.sleepBefore > 0 {
					time.Sleep(st.sleepBefore)
				}

				got := sw.Allow(st.key)

				if got.Allowed != st.wantAllowed {
					t.Errorf("step %d (key %q): Allowed = %v, want %v", i, st.key, got.Allowed, st.wantAllowed)
				}
				if got.Remaining != st.wantRemaining {
					t.Errorf("step %d (key %q): Remaining = %d, want %d", i, st.key, got.Remaining, st.wantRemaining)
				}
				if got.Limit != st.wantLimit {
					t.Errorf("step %d (key %q): Limit = %d, want %d", i, st.key, got.Limit, st.wantLimit)
				}
				if st.checkRetryAfter && got.RetryAfter <= 0 {
					t.Errorf("step %d (key %q): RetryAfter = %v, want > 0", i, st.key, got.RetryAfter)
				}
			}
		})
	}
}
