package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticVerifier(t *testing.T) {
	secret := "top-secret"
	v := &StaticVerifier{Secret: secret}
	ctx := context.Background()

	t.Run("valid_token", func(t *testing.T) {
		id, err := v.Verify(ctx, secret)
		assert.NoError(t, err)
		assert.Equal(t, "admin_user", id)
	})

	t.Run("invalid_token", func(t *testing.T) {
		id, err := v.Verify(ctx, "wrong-token")
		assert.Error(t, err)
		assert.Empty(t, id)
	})
}

func TestNoOpVerifier(t *testing.T) {
	v := &NoOpVerifier{}
	ctx := context.Background()

	t.Run("any_token_is_valid", func(t *testing.T) {
		id, err := v.Verify(ctx, "any-id")
		assert.NoError(t, err)
		assert.Equal(t, "any-id", id)
	})

	t.Run("empty_token_fails", func(t *testing.T) {
		_, err := v.Verify(ctx, "")
		assert.Error(t, err)
	})
}
