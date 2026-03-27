// Package analytics provides a real-time statistical engine for API request
// latency analysis. It uses goroutines for background sample decay and
// mutex-based synchronization for thread safety.
package analytics

import (
	"math"
	"sort"
	"sync"
	"time"
)

// Sample represents a single recorded latency measurement.
type Sample struct {
	Latency   float64   // Response time in milliseconds
	Timestamp time.Time // When the request was recorded
	ClientID  string    // Which client made the request
}

// Snapshot holds the computed statistics at a point in time.
type Snapshot struct {
	Mean           float64 `json:"mean_ms"`
	Median         float64 `json:"median_ms"`
	StdDev         float64 `json:"std_dev_ms"`
	P95            float64 `json:"p95_ms"`
	P99            float64 `json:"p99_ms"`
	Min            float64 `json:"min_ms"`
	Max            float64 `json:"max_ms"`
	TotalSamples   int     `json:"total_samples"`
	Throughput     float64 `json:"throughput_rps"`
	WindowSeconds  float64 `json:"window_seconds"`
	VarianceCoeff  float64 `json:"variance_coefficient"`
}

// Collector gathers latency samples and computes statistics on demand.
// It is safe for concurrent use by multiple goroutines.
type Collector struct {
	mu      sync.RWMutex
	samples []Sample
	window  time.Duration // Only consider samples within this window
	stop    chan struct{}
	done    chan struct{}
}

// NewCollector creates a Collector with a specified analysis window duration.
// Samples older than the window are periodically pruned by a background worker.
func NewCollector(window time.Duration) *Collector {
	c := &Collector{
		samples: make([]Sample, 0, 256),
		window:  window,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go c.startDecayWorker(10 * time.Second)
	return c
}

// Record adds a new latency sample to the collector.
func (c *Collector) Record(clientID string, latencyMs float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.samples = append(c.samples, Sample{
		Latency:   latencyMs,
		Timestamp: time.Now(),
		ClientID:  clientID,
	})
}

// Snapshot computes and returns current statistical metrics from active samples.
func (c *Collector) Snapshot() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cutoff := time.Now().Add(-c.window)
	active := make([]float64, 0, len(c.samples))
	var earliest time.Time

	for _, s := range c.samples {
		if s.Timestamp.After(cutoff) {
			active = append(active, s.Latency)
			if earliest.IsZero() || s.Timestamp.Before(earliest) {
				earliest = s.Timestamp
			}
		}
	}

	n := len(active)
	if n == 0 {
		return Snapshot{WindowSeconds: c.window.Seconds()}
	}

	// Sort for median and percentile calculations
	sorted := make([]float64, n)
	copy(sorted, active)
	sort.Float64s(sorted)

	mean := Mean(active)
	stddev := StdDev(active, mean)

	// Calculate throughput (requests per second)
	elapsed := time.Since(earliest).Seconds()
	if elapsed < 1 {
		elapsed = 1
	}
	throughput := float64(n) / elapsed

	// Coefficient of variation: stddev / mean (unitless measure of dispersion)
	varCoeff := 0.0
	if mean > 0 {
		varCoeff = math.Round((stddev/mean)*10000) / 10000
	}

	return Snapshot{
		Mean:          math.Round(mean*100) / 100,
		Median:        math.Round(Median(sorted)*100) / 100,
		StdDev:        math.Round(stddev*100) / 100,
		P95:           math.Round(Percentile(sorted, 95)*100) / 100,
		P99:           math.Round(Percentile(sorted, 99)*100) / 100,
		Min:           math.Round(sorted[0]*100) / 100,
		Max:           math.Round(sorted[n-1]*100) / 100,
		TotalSamples:  n,
		Throughput:    math.Round(throughput*100) / 100,
		WindowSeconds: c.window.Seconds(),
		VarianceCoeff: varCoeff,
	}
}

// Shutdown gracefully stops the background decay worker.
func (c *Collector) Shutdown() {
	select {
	case <-c.done:
		return
	default:
		close(c.stop)
		<-c.done
	}
}

// startDecayWorker periodically removes samples older than the analysis window.
func (c *Collector) startDecayWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(c.done)

	for {
		select {
		case <-ticker.C:
			c.decay()
		case <-c.stop:
			return
		}
	}
}

// decay removes expired samples from the collector's buffer.
func (c *Collector) decay() {
	cutoff := time.Now().Add(-c.window)
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find first valid index using binary-style scan
	kept := c.samples[:0]
	for _, s := range c.samples {
		if s.Timestamp.After(cutoff) {
			kept = append(kept, s)
		}
	}
	c.samples = kept
}

// ─── MATH FUNCTIONS ──────────────────────────────────────────────────────────

// Mean computes the arithmetic mean of a slice of float64 values.
func Mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

// Median returns the middle value of a pre-sorted slice.
// For even-length slices, it returns the average of the two middle values.
func Median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2.0
}

// StdDev computes the population standard deviation given data and its mean.
// Uses the formula: sqrt( Σ(xi - mean)² / N )
func StdDev(data []float64, mean float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sumSqDiff := 0.0
	for _, v := range data {
		diff := v - mean
		sumSqDiff += diff * diff
	}
	return math.Sqrt(sumSqDiff / float64(len(data)))
}

// Percentile computes the p-th percentile of a pre-sorted slice using
// linear interpolation between closest ranks.
func Percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}

	// Rank calculation using linear interpolation
	rank := (p / 100.0) * float64(n-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	fraction := rank - float64(lower)

	if upper >= n {
		upper = n - 1
	}

	return sorted[lower] + fraction*(sorted[upper]-sorted[lower])
}
