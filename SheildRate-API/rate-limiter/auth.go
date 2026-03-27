package ratelimiter

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Hash password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Check password
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	)
}

// Register client
func (rl *RateLimiter) RegisterClient(clientID, password string) error {

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	rl.Mutex.Lock()
	defer rl.Mutex.Unlock()

	now := time.Now()

	rl.Clients[clientID] = &Client{
		Metadata:     Metadata{ClientID: clientID},
		RequestCount: 0,
		WindowStart:  now,
		LastSeen:     now,
		RequestLog:   make([]time.Time, 0),
		PasswordHash: hash,
	}

	return nil
}

// Authenticate client
func (rl *RateLimiter) Authenticate(clientID, password string) error {

	rl.Mutex.RLock()
	client, exists := rl.Clients[clientID]
	rl.Mutex.RUnlock()

	if !exists {
		return bcrypt.ErrMismatchedHashAndPassword
	}

	err := CheckPassword(client.PasswordHash, password)
	if err != nil {
		return err
	}

	rl.Mutex.Lock()
	if existingClient, ok := rl.Clients[clientID]; ok {
		existingClient.LastSeen = time.Now()
	}
	rl.Mutex.Unlock()

	return nil
}
