package algorithms

import (
	"testing"
	"time"
)

func TestFixedWindow_Allow(t *testing.T) {
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
			name:   "1. First request for a new key is always allowed",
			limit:  5,
			window: 1 * time.Second,
			steps: []step{
				{
					key:           "192.168.1.1",
					wantAllowed:   true,
					wantRemaining: 4,
					wantLimit:     5,
				},
			},
		},
		{
			name:   "2. Requests up to limit are allowed and remaining count decrements correctly",
			limit:  3,
			window: 1 * time.Second,
			steps: []step{
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 2, wantLimit: 3},
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 1, wantLimit: 3},
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 0, wantLimit: 3},
			},
		},
		{
			name:   "3. Request exceeding limit is denied with Allowed=false and RetryAfter > 0",
			limit:  2,
			window: 500 * time.Millisecond,
			steps: []step{
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{
					key:             "192.168.1.1",
					wantAllowed:     false,
					wantRemaining:   0,
					wantLimit:       2,
					checkRetryAfter: true,
				},
			},
		},
		{
			name:   "4. After window expires counter resets and new request is allowed",
			limit:  2,
			window: 50 * time.Millisecond,
			steps: []step{
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{key: "192.168.1.1", wantAllowed: false, wantRemaining: 0, wantLimit: 2, checkRetryAfter: true},
				{
					key:           "192.168.1.1",
					sleepBefore:   60 * time.Millisecond,
					wantAllowed:   true,
					wantRemaining: 1,
					wantLimit:     2,
				},
			},
		},
		{
			name:   "5. Two different keys (IPs) are tracked independently",
			limit:  2,
			window: 1 * time.Second,
			steps: []step{
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "192.168.1.1", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{key: "192.168.1.1", wantAllowed: false, wantRemaining: 0, wantLimit: 2},
				// 192.168.1.2 is tracked independently
				{key: "192.168.1.2", wantAllowed: true, wantRemaining: 1, wantLimit: 2},
				{key: "192.168.1.2", wantAllowed: true, wantRemaining: 0, wantLimit: 2},
				{key: "192.168.1.2", wantAllowed: false, wantRemaining: 0, wantLimit: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := NewFixedWindow(tt.limit, tt.window)

			for i, st := range tt.steps {
				if st.sleepBefore > 0 {
					time.Sleep(st.sleepBefore)
				}

				got := fw.Allow(st.key)

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
