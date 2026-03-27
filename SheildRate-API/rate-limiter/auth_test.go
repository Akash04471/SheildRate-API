package ratelimiter

import "testing"

func TestPasswordHashing(t *testing.T) {

	password := "secure123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Hashing failed: %v", err)
	}

	err = CheckPassword(hash, password)
	if err != nil {
		t.Errorf("Password should match")
	}

	err = CheckPassword(hash, "wrong")
	if err == nil {
		t.Errorf("Password should not match")
	}
}
