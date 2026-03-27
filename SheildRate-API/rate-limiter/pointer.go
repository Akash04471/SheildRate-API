package ratelimiter

import "fmt"

// Call by value
func IncrementByValue(count int) {
	count++
	fmt.Println("Inside IncrementByValue:", count)
}

// Call by pointer (reference)
func IncrementByPointer(count *int) {
	*count++
	fmt.Println("Inside IncrementByPointer:", *count)
}
