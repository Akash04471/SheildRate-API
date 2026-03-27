package ratelimiter

import "encoding/json"

// Marshal struct to JSON
func ToJSON(client Client) ([]byte, error) {
	return json.Marshal(client)
}

// Unmarshal JSON to struct
func FromJSON(data []byte) (Client, error) {
	var client Client
	err := json.Unmarshal(data, &client)
	return client, err
}
