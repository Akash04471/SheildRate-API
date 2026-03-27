package main

import (
	"fmt"
	"sync"
	"time"

	ratelimiter "api-rate-limiter/rate-limiter"
)

type requestResult struct {
	clientID  string
	requestNo int
	allowed   bool
	err       error
}

// This lab demo shows fan-out/fan-in concurrency with goroutines and channels.
func main() {
	maxRequests := 5
	window := 2 * time.Second
	requestsPerClient := 8
	clients := []string{"client-A", "client-B", "client-C"}

	fmt.Println("=== Goroutine Lab: Concurrency Demonstration ===")
	fmt.Printf("Config -> maxRequests=%d, window=%s, requestsPerClient=%d\n", maxRequests, window, requestsPerClient)
	fmt.Println()

	// Baseline run without goroutines (sequential processing).
	sequential := runSimulation(false, maxRequests, window, requestsPerClient, clients)
	printSummary("Sequential", sequential)

	fmt.Println()

	// Concurrency run using goroutines + WaitGroup + channels.
	concurrent := runSimulation(true, maxRequests, window, requestsPerClient, clients)
	printSummary("Concurrent", concurrent)

	fmt.Println()
	fmt.Printf("Time comparison -> sequential=%s, concurrent=%s\n", sequential.elapsed, concurrent.elapsed)
	if concurrent.elapsed < sequential.elapsed {
		fmt.Println("Observation: concurrent execution is faster for this simulated workload.")
	} else {
		fmt.Println("Observation: timing can vary, but goroutines allow many requests to progress together.")
	}

	fmt.Println("Key idea: fan-out starts request goroutines, fan-in collects all results through one channel.")
}

type simulationSummary struct {
	elapsed      time.Duration
	allowedTotal int
	blockedTotal int
	byClient     map[string]struct {
		allowed int
		blocked int
	}
}

func runSimulation(concurrent bool, maxRequests int, window time.Duration, requestsPerClient int, clients []string) simulationSummary {
	rl := ratelimiter.NewRateLimiter()
	defer rl.Shutdown()

	var wg sync.WaitGroup
	results := make(chan requestResult, len(clients)*requestsPerClient)

	start := time.Now()
	byClient := make(map[string]struct {
		allowed int
		blocked int
	})

	for _, clientID := range clients {
		for req := 1; req <= requestsPerClient; req++ {
			if concurrent {
				wg.Add(1)

				// Snippet 1 (fan-out): one goroutine handles one request.
				go func(cid string, requestNo int) {
					defer wg.Done()
					allowed, err := rl.Allow(cid, maxRequests, window)
					results <- requestResult{clientID: cid, requestNo: requestNo, allowed: allowed, err: err}
				}(clientID, req)
			} else {
				allowed, err := rl.Allow(clientID, maxRequests, window)
				results <- requestResult{clientID: clientID, requestNo: req, allowed: allowed, err: err}
			}
		}
	}

	if concurrent {
		// Snippet 2 (fan-in boundary): close results after all goroutines complete.
		go func() {
			wg.Wait()
			close(results)
		}()
	} else {
		close(results)
	}

	allowedTotal := 0
	blockedTotal := 0

	for result := range results {
		if result.err != nil {
			fmt.Printf("client=%s request=%d error=%v\n", result.clientID, result.requestNo, result.err)
			continue
		}

		status := "BLOCKED"
		counts := byClient[result.clientID]
		if result.allowed {
			status = "ALLOWED"
			allowedTotal++
			counts.allowed++
		} else {
			blockedTotal++
			counts.blocked++
		}
		byClient[result.clientID] = counts

		fmt.Printf("client=%s request=%d status=%s\n", result.clientID, result.requestNo, status)
	}

	elapsed := time.Since(start)

	return simulationSummary{
		elapsed:      elapsed,
		allowedTotal: allowedTotal,
		blockedTotal: blockedTotal,
		byClient:     byClient,
	}
}

func printSummary(label string, summary simulationSummary) {
	fmt.Printf("=== %s Summary ===\n", label)
	fmt.Printf("allowed=%d blocked=%d elapsed=%s\n", summary.allowedTotal, summary.blockedTotal, summary.elapsed)
	for clientID, counts := range summary.byClient {
		fmt.Printf("%s -> allowed=%d blocked=%d\n", clientID, counts.allowed, counts.blocked)
	}

	fmt.Println("Expected behavior: for each client, first 5 requests are ALLOWED and remaining are BLOCKED in the 2s window.")
}
