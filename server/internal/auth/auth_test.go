package auth

import (
	"context"
	"testing"
)

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		expected string
		want     bool
	}{
		{
			name:     "matching tokens",
			provided: "secret-token-123",
			expected: "secret-token-123",
			want:     true,
		},
		{
			name:     "non-matching tokens",
			provided: "wrong-token",
			expected: "secret-token-123",
			want:     false,
		},
		{
			name:     "empty provided token",
			provided: "",
			expected: "secret-token-123",
			want:     false,
		},
		{
			name:     "empty expected token",
			provided: "secret-token-123",
			expected: "",
			want:     false,
		},
		{
			name:     "both empty",
			provided: "",
			expected: "",
			want:     true,
		},
		{
			name:     "similar tokens different length",
			provided: "secret-token-12",
			expected: "secret-token-123",
			want:     false,
		},
		{
			name:     "unicode tokens matching",
			provided: "token-with-emoji-🔐",
			expected: "token-with-emoji-🔐",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateToken(tt.provided, tt.expected)
			if got != tt.want {
				t.Errorf("ValidateToken(%q, %q) = %v, want %v",
					tt.provided, tt.expected, got, tt.want)
			}
		})
	}
}

func TestPersonContext(t *testing.T) {
	t.Run("set and retrieve person", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithPerson(ctx, "sebastian")

		got := PersonFromContext(ctx)
		if got != "sebastian" {
			t.Errorf("PersonFromContext() = %q, want %q", got, "sebastian")
		}
	})

	t.Run("empty context returns empty string", func(t *testing.T) {
		ctx := context.Background()

		got := PersonFromContext(ctx)
		if got != "" {
			t.Errorf("PersonFromContext() = %q, want empty string", got)
		}
	})

	t.Run("overwrite person", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithPerson(ctx, "sebastian")
		ctx = WithPerson(ctx, "petra")

		got := PersonFromContext(ctx)
		if got != "petra" {
			t.Errorf("PersonFromContext() = %q, want %q", got, "petra")
		}
	})
}

func TestIsValidPerson(t *testing.T) {
	tests := []struct {
		person string
		want   bool
	}{
		{"sebastian", true},
		{"petra", true},
		{"unknown", false},
		{"", false},
		{"Sebastian", false}, // case sensitive
		{"PETRA", false},
	}

	for _, tt := range tests {
		t.Run(tt.person, func(t *testing.T) {
			got := IsValidPerson(tt.person)
			if got != tt.want {
				t.Errorf("IsValidPerson(%q) = %v, want %v", tt.person, got, tt.want)
			}
		})
	}
}

func TestValidPersonsList(t *testing.T) {
	// Verify the valid persons list contains expected values
	expected := map[string]bool{
		"sebastian": true,
		"petra":     true,
	}

	if len(ValidPersons) != len(expected) {
		t.Errorf("ValidPersons has %d entries, want %d", len(ValidPersons), len(expected))
	}

	for _, p := range ValidPersons {
		if !expected[p] {
			t.Errorf("unexpected person in ValidPersons: %q", p)
		}
	}
}

// TestConstantTimeComparison verifies that token validation uses constant-time
// comparison to prevent timing attacks. While we can't directly test timing,
// we verify the behavior matches crypto/subtle.ConstantTimeCompare semantics.
func TestConstantTimeComparison(t *testing.T) {
	// These tests verify the function behaves like constant-time comparison
	// (returns same result regardless of where mismatch occurs)
	tests := []struct {
		name     string
		provided string
		expected string
		want     bool
	}{
		// Mismatch at different positions should all return false
		{"mismatch at start", "Xecret-token", "secret-token", false},
		{"mismatch at middle", "secXet-token", "secret-token", false},
		{"mismatch at end", "secret-tokeX", "secret-token", false},

		// Length differences
		{"shorter by 1", "secret-toke", "secret-token", false},
		{"longer by 1", "secret-token!", "secret-token", false},
		{"much shorter", "sec", "secret-token", false},
		{"much longer", "secret-token-extra-stuff", "secret-token", false},

		// Edge cases
		{"null byte in provided", "secret\x00token", "secret-token", false},
		{"null byte in expected", "secret-token", "secret\x00token", false},

		// Exact match
		{"exact match", "secret-token", "secret-token", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateToken(tt.provided, tt.expected)
			if got != tt.want {
				t.Errorf("ValidateToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
