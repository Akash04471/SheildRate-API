package ratelimiter

// GetClientID extracts a client identifier from a remote address.
func GetClientID(remoteAddr string) string {
	return remoteAddr
}
