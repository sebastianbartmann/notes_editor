// Package auth provides authentication and person context management.
package auth

import (
	"context"
	"crypto/subtle"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// personContextKey is the context key for storing the person value.
	personContextKey contextKey = "person"
)

// ValidateToken performs constant-time comparison of the provided token
// against the expected token to prevent timing attacks.
func ValidateToken(provided, expected string) bool {
	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// PersonFromContext retrieves the person value from the context.
// Returns empty string if no person is set.
func PersonFromContext(ctx context.Context) string {
	person, ok := ctx.Value(personContextKey).(string)
	if !ok {
		return ""
	}
	return person
}

// WithPerson returns a new context with the person value set.
func WithPerson(ctx context.Context, person string) context.Context {
	return context.WithValue(ctx, personContextKey, person)
}
