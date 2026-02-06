// Package auth provides interfaces and implementations for user authentication
// and token verification within the dashboard service.
//
// It abstracts the authentication logic, allowing the business logic to remain
// agnostic of the specific token structure (e.g., JWT, Static Key, or Opaque tokens).
package auth

import (
	"context"
	"errors"
)

// Verifier defines the core contract for validating authentication tokens.
// Custom authentication strategies (like JWT, OAuth2, or Introspection)
// should implement this interface to be compatible with the service.
type Verifier interface {
	// Verify checks the validity of the provided raw token string.
	// On success, it returns the unique identifier (memberID) of the user.
	// On failure, it returns an empty string and a descriptive error.
	Verify(ctx context.Context, token string) (string, error)
}

// StaticVerifier implements the Verifier interface using a simple pre-defined secret.
// This is primarily used for system-to-service communication or administrative access.
type StaticVerifier struct {
	// Secret is the pre-configured key that must match the incoming token.
	Secret string
}

// Verify checks if the token matches the configured Secret.
// If it matches, it returns "admin_user" as the authenticated identity.
func (s *StaticVerifier) Verify(ctx context.Context, token string) (string, error) {
	// Compare incoming token with the pre-shared secret
	if token == s.Secret {
		return "admin_user", nil
	}
	return "", errors.New("invalid static token")
}

// NoOpVerifier is a pass-through implementation of the Verifier interface.
// It treats the provided token string itself as the user identity.
//
// WARNING: This implementation provides NO security and should only be used
// for local development or testing purposes.
type NoOpVerifier struct{}

// Verify ensures the token is not empty and returns it as the member ID.
// It assumes that the client is providing their own identity in place of a token.
func (n *NoOpVerifier) Verify(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", errors.New("empty token")
	}
	// In No-Op mode, the token string is treated directly as the UserID.
	return token, nil
}
