package auth

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// StaticVerifier Tests
// ============================================================================

func TestStaticVerifier_Verify(t *testing.T) {
	secret := "top-secret"
	v := &StaticVerifier{Secret: secret}
	ctx := context.Background()

	tests := []struct {
		name       string
		token      string
		wantUserID string
		wantErr    error
	}{
		{
			name:       "valid_token",
			token:      secret,
			wantUserID: "admin_user",
			wantErr:    nil,
		},
		{
			name:       "invalid_token",
			token:      "wrong-token",
			wantUserID: "",
			wantErr:    ErrInvalidToken,
		},
		{
			name:       "empty_token",
			token:      "",
			wantUserID: "",
			wantErr:    ErrEmptyToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := v.Verify(ctx, tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, userID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUserID, userID)
			}
		})
	}
}

// ============================================================================
// NoOpVerifier Tests
// ============================================================================

func TestNoOpVerifier_Verify(t *testing.T) {
	v := &NoOpVerifier{}
	ctx := context.Background()

	tests := []struct {
		name       string
		token      string
		wantUserID string
		wantErr    error
	}{
		{
			name:       "valid_token",
			token:      "user123",
			wantUserID: "user123",
			wantErr:    nil,
		},
		{
			name:       "another_valid_token",
			token:      "another-user-456",
			wantUserID: "another-user-456",
			wantErr:    nil,
		},
		{
			name:       "empty_token",
			token:      "",
			wantUserID: "",
			wantErr:    ErrEmptyToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := v.Verify(ctx, tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, userID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUserID, userID)
			}
		})
	}
}

// ============================================================================
// JWTVerifier Tests
// ============================================================================

type jwtVerifyTestCase struct {
	name       string
	token      string
	wantUserID string
	wantErr    error
}

func TestJWTVerifier_Verify(t *testing.T) {
	secret := []byte("test-secret-key")
	verifier := NewJWTVerifier(secret)
	ctx := context.Background()

	createToken := func(claims jwt.MapClaims, signingMethod jwt.SigningMethod) string {
		token := jwt.NewWithClaims(signingMethod, claims)
		tokenString, _ := token.SignedString(secret)
		return tokenString
	}

	createInvalidToken := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("wrong-secret"))
		return tokenString
	}

	tests := buildJWTVerifyCases(createToken, createInvalidToken)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertJWTVerifyResult(t, verifier, ctx, tt)
		})
	}
}

func buildJWTVerifyCases(
	createToken func(jwt.MapClaims, jwt.SigningMethod) string,
	createInvalidToken func(jwt.MapClaims) string,
) []jwtVerifyTestCase {
	return append(
		buildJWTVerifyBaseCases(createToken, createInvalidToken),
		buildJWTVerifySigningCases(createToken)...,
	)
}

func buildJWTVerifyBaseCases(
	createToken func(jwt.MapClaims, jwt.SigningMethod) string,
	createInvalidToken func(jwt.MapClaims) string,
) []jwtVerifyTestCase {
	return append(
		buildJWTVerifyValidCases(createToken),
		buildJWTVerifyInvalidCases(createInvalidToken)...,
	)
}

func buildJWTVerifyValidCases(
	createToken func(jwt.MapClaims, jwt.SigningMethod) string,
) []jwtVerifyTestCase {
	return []jwtVerifyTestCase{
		{
			name: "valid_token_with_user_id",
			token: createToken(jwt.MapClaims{
				"user_id": "user123",
				"exp":     time.Now().Add(time.Hour).Unix(),
			}, jwt.SigningMethodHS256),
			wantUserID: "user123",
			wantErr:    nil,
		},
		{
			name: "valid_token_without_expiration",
			token: createToken(jwt.MapClaims{
				"user_id": "user456",
			}, jwt.SigningMethodHS256),
			wantUserID: "user456",
			wantErr:    nil,
		},
		{
			name: "expired_token",
			token: createToken(jwt.MapClaims{
				"user_id": "user789",
				"exp":     time.Now().Add(-time.Hour).Unix(),
			}, jwt.SigningMethodHS256),
			wantUserID: "",
			wantErr:    ErrTokenExpired,
		},
		{
			name: "missing_user_id_claim",
			token: createToken(jwt.MapClaims{
				"username": "john",
			}, jwt.SigningMethodHS256),
			wantUserID: "",
			wantErr:    ErrMissingUserID,
		},
		{
			name: "empty_user_id_claim",
			token: createToken(jwt.MapClaims{
				"user_id": "",
			}, jwt.SigningMethodHS256),
			wantUserID: "",
			wantErr:    ErrMissingUserID,
		},
	}
}

func buildJWTVerifyInvalidCases(
	createInvalidToken func(jwt.MapClaims) string,
) []jwtVerifyTestCase {
	return []jwtVerifyTestCase{
		{
			name: "invalid_signature",
			token: createInvalidToken(jwt.MapClaims{
				"user_id": "hacker",
			}),
			wantUserID: "",
			wantErr:    ErrInvalidToken,
		},
		{
			name:       "empty_token",
			token:      "",
			wantUserID: "",
			wantErr:    ErrEmptyToken,
		},
		{
			name:       "malformed_token",
			token:      malformedToken(),
			wantUserID: "",
			wantErr:    ErrInvalidToken,
		},
	}
}

func buildJWTVerifySigningCases(
	createToken func(jwt.MapClaims, jwt.SigningMethod) string,
) []jwtVerifyTestCase {
	return []jwtVerifyTestCase{
		{
			name: "wrong_signing_method",
			token: createToken(jwt.MapClaims{
				"user_id": "user999",
			}, jwt.SigningMethodHS512),
			wantUserID: "",
			wantErr:    ErrInvalidToken,
		},
	}
}

func malformedToken() string {
	return strings.Repeat("invalid", 2)
}

func assertJWTVerifyResult(t *testing.T, verifier *JWTVerifier, ctx context.Context, tt jwtVerifyTestCase) {
	t.Helper()

	userID, err := verifier.Verify(ctx, tt.token)
	if tt.wantErr != nil {
		assert.Error(t, err)
		if tt.name == "wrong_signing_method" {
			assert.ErrorIs(t, err, ErrInvalidToken)
			assert.Contains(t, err.Error(), "signing method")
		} else {
			assert.ErrorIs(t, err, tt.wantErr)
		}
		assert.Empty(t, userID)
		return
	}

	assert.NoError(t, err)
	assert.Equal(t, tt.wantUserID, userID)
}

// Test different signing methods.
func TestJWTVerifier_DifferentSigningMethods(t *testing.T) {
	secret := []byte("test-secret")
	ctx := context.Background()

	testCases := []struct {
		name   string
		method jwt.SigningMethod
	}{
		{"HS256", jwt.SigningMethodHS256},
		{"HS384", jwt.SigningMethodHS384},
		{"HS512", jwt.SigningMethodHS512},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			verifier := NewJWTVerifierWithMethod(secret, tc.method)

			// Create token with the same method
			token := jwt.NewWithClaims(tc.method, jwt.MapClaims{
				"user_id": "test-user",
			})
			tokenString, err := token.SignedString(secret)
			require.NoError(t, err)

			// Verify should succeed
			userID, err := verifier.Verify(ctx, tokenString)
			assert.NoError(t, err)
			assert.Equal(t, "test-user", userID)
		})
	}
}

// Test constructors.
func TestNewJWTVerifier(t *testing.T) {
	secret := []byte("my-secret")
	verifier := NewJWTVerifier(secret)

	assert.NotNil(t, verifier)
	assert.Equal(t, secret, verifier.Secret)
	assert.Equal(t, jwt.SigningMethodHS256, verifier.SigningMethod)
}

func TestNewJWTVerifierWithMethod(t *testing.T) {
	secret := []byte("my-secret")
	method := jwt.SigningMethodHS512
	verifier := NewJWTVerifierWithMethod(secret, method)

	assert.NotNil(t, verifier)
	assert.Equal(t, secret, verifier.Secret)
	assert.Equal(t, method, verifier.SigningMethod)
}

// Test context cancellation.
func TestJWTVerifier_ContextCancellation(t *testing.T) {
	secret := []byte("test-secret")
	verifier := NewJWTVerifier(secret)

	// Create a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user123",
	})
	tokenString, _ := token.SignedString(secret)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Verify should still work (JWT parsing doesn't check context)
	// But in real implementation, you might want to add context checks
	userID, err := verifier.Verify(ctx, tokenString)
	assert.NoError(t, err)
	assert.Equal(t, "user123", userID)
}

// Test edge cases.
func TestJWTVerifier_EdgeCases(t *testing.T) {
	secret := []byte("test-secret")
	verifier := NewJWTVerifier(secret)
	ctx := context.Background()

	t.Run("token_with_null_user_id", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": nil,
		})
		tokenString, _ := token.SignedString(secret)

		userID, err := verifier.Verify(ctx, tokenString)
		assert.ErrorIs(t, err, ErrMissingUserID)
		assert.Empty(t, userID)
	})

	t.Run("token_with_numeric_user_id", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": 12345, // Number instead of string
		})
		tokenString, _ := token.SignedString(secret)

		userID, err := verifier.Verify(ctx, tokenString)
		assert.ErrorIs(t, err, ErrMissingUserID)
		assert.Empty(t, userID)
	})

	t.Run("token_with_future_expiration", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": "user123",
			"exp":     time.Now().Add(24 * time.Hour).Unix(),
		})
		tokenString, _ := token.SignedString(secret)

		userID, err := verifier.Verify(ctx, tokenString)
		assert.NoError(t, err)
		assert.Equal(t, "user123", userID)
	})
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkStaticVerifier_Verify(b *testing.B) {
	v := &StaticVerifier{Secret: "secret"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.Verify(ctx, "secret")
	}
}

func BenchmarkNoOpVerifier_Verify(b *testing.B) {
	v := &NoOpVerifier{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.Verify(ctx, "user123")
	}
}

func BenchmarkJWTVerifier_Verify(b *testing.B) {
	secret := []byte("benchmark-secret")
	verifier := NewJWTVerifier(secret)
	ctx := context.Background()

	// Create a valid token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "bench-user",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(secret)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.Verify(ctx, tokenString)
	}
}

// ============================================================================
// Example Tests
// ============================================================================

func ExampleStaticVerifier() {
	verifier := &StaticVerifier{Secret: "my-secret-key"}
	ctx := context.Background()

	userID, err := verifier.Verify(ctx, "my-secret-key")
	if err != nil {
		return
	}

	fmt.Println(userID)
	// Output: admin_user
}

func ExampleJWTVerifier() {
	secret := []byte("jwt-secret")
	verifier := NewJWTVerifier(secret)
	ctx := context.Background()

	// Create a token (in real app, this comes from the client)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user123",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(secret)

	// Verify the token
	userID, err := verifier.Verify(ctx, tokenString)
	if err != nil {
		return
	}

	fmt.Println(userID)
	// Output: user123
}

func ExampleNoOpVerifier() {
	verifier := &NoOpVerifier{}
	ctx := context.Background()

	// In NoOp mode, the token itself is treated as the user ID
	userID, err := verifier.Verify(ctx, "user123")
	if err != nil {
		return
	}

	fmt.Println(userID)
	// Output: user123
}

func ExampleNewJWTVerifier() {
	// Create a JWT verifier with HS256 signing method
	secret := []byte("my-secret-key")
	verifier := NewJWTVerifier(secret)

	ctx := context.Background()

	// Generate a token for testing
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "alice",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(secret)

	// Verify the token
	userID, err := verifier.Verify(ctx, tokenString)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(userID)
	// Output: alice
}

func ExampleNewJWTVerifierWithMethod() {
	// Create a JWT verifier with custom signing method (HS512)
	secret := []byte("my-secret-key")
	verifier := NewJWTVerifierWithMethod(secret, jwt.SigningMethodHS512)

	ctx := context.Background()

	// Generate a token with HS512
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"user_id": "bob",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(secret)

	// Verify the token
	userID, err := verifier.Verify(ctx, tokenString)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(userID)
	// Output: bob
}
