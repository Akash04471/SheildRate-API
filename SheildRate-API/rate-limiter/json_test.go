package ratelimiter

import "testing"

func TestJSONMarshalUnmarshal(t *testing.T) {

	original := Client{
		RequestCount: 10,
	}

	data, err := ToJSON(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	result, err := FromJSON(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.RequestCount != original.RequestCount {
		t.Errorf("Expected %d, got %d",
			original.RequestCount,
			result.RequestCount,
		)
	}
}
