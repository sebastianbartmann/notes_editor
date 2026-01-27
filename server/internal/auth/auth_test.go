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
