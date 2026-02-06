// Package auth provides interfaces and implementations for user authentication
// and token verification within the dashboard service.
//
// It abstracts the authentication logic, allowing the business logic to remain
// agnostic of the specific token structure (e.g., JWT, Static Key, or Opaque tokens).
//
// Example usage:
//
//	// Using JWT Verifier
//	verifier := &auth.JWTVerifier{Secret: []byte("my-secret-key")}
//	userID, err := verifier.Verify(ctx, tokenString)
//	if err != nil {
//	    // Handle authentication error
//	}
//
//	// Using Static Verifier for service-to-service communication
//	verifier := &auth.StaticVerifier{Secret: "shared-secret"}
//	userID, err := verifier.Verify(ctx, token)
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// Common authentication errors
var (
	// ErrInvalidToken is returned when the token format is invalid or cannot be parsed.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired is returned when the JWT token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrEmptyToken is returned when an empty token string is provided.
	ErrEmptyToken = errors.New("empty token")

	// ErrMissingUserID is returned when the token doesn't contain a user_id claim.
	ErrMissingUserID = errors.New("missing user_id claim in token")

	// ErrInvalidSigningMethod is returned when JWT uses an unexpected signing algorithm.
	ErrInvalidSigningMethod = errors.New("invalid token signing method")
)

// Verifier defines the core contract for validating authentication tokens.
// Custom authentication strategies (like JWT, OAuth2, or Introspection)
// should implement this interface to be compatible with the service.
//
// All implementations must be safe for concurrent use by multiple goroutines.
type Verifier interface {
	// Verify checks the validity of the provided raw token string.
	// On success, it returns the unique identifier (memberID) of the user.
	// On failure, it returns an empty string and a descriptive error.
	//
	// The context can be used for timeout control and cancellation.
	Verify(ctx context.Context, token string) (string, error)
}

// StaticVerifier implements the Verifier interface using a simple pre-defined secret.
// This is primarily used for system-to-service communication or administrative access.
//
// Security Note: StaticVerifier provides basic authentication suitable only for
// trusted internal services. For user-facing authentication, use JWTVerifier instead.
//
// Example:
//
//	verifier := &StaticVerifier{Secret: "my-shared-secret"}
//	userID, err := verifier.Verify(ctx, "my-shared-secret")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Authenticated as:", userID) // Output: "admin_user"
type StaticVerifier struct {
	// Secret is the pre-configured key that must match the incoming token.
	// This should be kept confidential and rotated regularly.
	Secret string
}

// Verify checks if the token matches the configured Secret.
// If it matches, it returns "admin_user" as the authenticated identity.
//
// This method is safe for concurrent use.
func (s *StaticVerifier) Verify(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrEmptyToken
	}

	if token == s.Secret {
		return "admin_user", nil
	}
	return "", ErrInvalidToken
}

// NoOpVerifier is a pass-through implementation of the Verifier interface.
// It treats the provided token string itself as the user identity.
//
// WARNING: This implementation provides NO security and should only be used
// for local development or testing purposes. Never use in production.
//
// Example:
//
//	verifier := &NoOpVerifier{}
//	userID, err := verifier.Verify(ctx, "user123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("User ID:", userID) // Output: "user123"
type NoOpVerifier struct{}

// Verify ensures the token is not empty and returns it as the member ID.
// It assumes that the client is providing their own identity in place of a token.
//
// This method is safe for concurrent use.
func (n *NoOpVerifier) Verify(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrEmptyToken
	}
	// In No-Op mode, the token string is treated directly as the UserID.
	return token, nil
}

// JWTVerifier implements the Verifier interface using JSON Web Tokens (JWT).
// It validates JWT signatures and extracts the user_id claim from the token payload.
//
// The verifier expects tokens to contain a "user_id" claim (string) that uniquely
// identifies the authenticated user.
//
// Supported signing methods: HS256, HS384, HS512 (HMAC with SHA-2)
//
// Example token claims:
//
//	{
//	  "user_id": "user123",
//	  "exp": 1735689600,
//	  "iat": 1735603200
//	}
//
// Usage:
//
//	verifier := &JWTVerifier{
//	    Secret:        []byte("my-secret-key"),
//	    SigningMethod: jwt.SigningMethodHS256, // Optional, defaults to HS256
//	}
//	userID, err := verifier.Verify(ctx, jwtTokenString)
//	if err != nil {
//	    log.Printf("Auth failed: %v", err)
//	    return
//	}
//	fmt.Printf("Authenticated user: %s\n", userID)
type JWTVerifier struct {
	// Secret is the key used to verify the JWT signature.
	// Must match the secret used to sign the tokens.
	Secret []byte

	// SigningMethod specifies the expected JWT signing algorithm.
	// If nil, defaults to HS256 (HMAC-SHA256).
	// Supported: HS256, HS384, HS512
	SigningMethod jwt.SigningMethod
}

// Verify validates a JWT token and extracts the user_id claim.
//
// The method performs the following checks:
//  1. Token format validation
//  2. Signature verification using the configured Secret
//  3. Expiration time validation (if "exp" claim is present)
//  4. Signing method validation
//  5. Extraction of "user_id" claim
//
// Returns the user ID on success, or an error if validation fails.
// This method is safe for concurrent use.
//
// Common errors:
//   - ErrEmptyToken: token string is empty
//   - ErrInvalidToken: malformed token or invalid signature
//   - ErrTokenExpired: token has passed its expiration time
//   - ErrMissingUserID: token doesn't contain user_id claim
//   - ErrInvalidSigningMethod: unexpected signing algorithm
func (j *JWTVerifier) Verify(ctx context.Context, tokenString string) (string, error) {
	if tokenString == "" {
		return "", ErrEmptyToken
	}

	expectedMethod := j.SigningMethod
	if expectedMethod == nil {
		expectedMethod = jwt.SigningMethodHS256
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != expectedMethod.Alg() {
			return nil, fmt.Errorf("%w: expected %s, got %s",
				ErrInvalidSigningMethod, expectedMethod.Alg(), token.Method.Alg())
		}
		return j.Secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrTokenExpired
		}
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("%w: failed to parse claims", ErrInvalidToken)
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return "", ErrMissingUserID
	}

	return userID, nil
}

// NewJWTVerifier creates a new JWTVerifier with the provided secret.
// It uses HS256 (HMAC-SHA256) as the default signing method.
//
// Example:
//
//	verifier := auth.NewJWTVerifier([]byte("my-secret-key"))
//	userID, err := verifier.Verify(ctx, token)
func NewJWTVerifier(secret []byte) *JWTVerifier {
	return &JWTVerifier{
		Secret:        secret,
		SigningMethod: jwt.SigningMethodHS256,
	}
}

// NewJWTVerifierWithMethod creates a new JWTVerifier with a custom signing method.
//
// Supported methods:
//   - jwt.SigningMethodHS256 (HMAC-SHA256) - Recommended
//   - jwt.SigningMethodHS384 (HMAC-SHA384)
//   - jwt.SigningMethodHS512 (HMAC-SHA512)
//
// Example:
//
//	verifier := auth.NewJWTVerifierWithMethod(
//	    []byte("my-secret-key"),
//	    jwt.SigningMethodHS512,
//	)
func NewJWTVerifierWithMethod(secret []byte, method jwt.SigningMethod) *JWTVerifier {
	return &JWTVerifier{
		Secret:        secret,
		SigningMethod: method,
	}
}
