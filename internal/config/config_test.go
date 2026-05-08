package config

import "testing"

func TestParseSize(t *testing.T) {
	tests := map[string]int64{
		"10":   10,
		"1KB":  1024,
		"2MB":  2 * 1024 * 1024,
		"500B": 500,
	}

	for input, want := range tests {
		got, err := parseSize(input)
		if err != nil {
			t.Fatalf("parseSize(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("parseSize(%q) = %d, want %d", input, got, want)
		}
	}
}
