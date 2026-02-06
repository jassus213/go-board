package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockAuth struct{ token string }

func (m *mockAuth) Verify(ctx context.Context, t string) (string, error) {
	if t == m.token {
		return "user1", nil
	}
	return "", assert.AnError
}

func TestAuthInterceptor(t *testing.T) {
	verifier := &mockAuth{token: "valid-pass"}
	interceptor := AuthInterceptor(verifier)

	t.Run("missing_metadata", func(t *testing.T) {
		err := interceptor(nil, &mockStream{ctx: context.Background()}, nil, nil)
		assert.ErrorContains(t, err, "missing metadata")
	})

	t.Run("valid_token", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer valid-pass")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		handler := func(srv interface{}, stream grpc.ServerStream) error {
			id := stream.Context().Value("member_id")
			assert.Equal(t, "user1", id)
			return nil
		}

		err := interceptor(nil, &mockStream{ctx: ctx}, nil, handler)
		assert.NoError(t, err)
	})
}

func TestAuthInterceptor_Fail(t *testing.T) {
	verifier := &mockAuth{token: "correct"}
	interceptor := AuthInterceptor(verifier)

	t.Run("invalid_token", func(t *testing.T) {
		md := metadata.Pairs("authorization", "wrong")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		err := interceptor(nil, &mockStream{ctx: ctx}, nil, nil)
		assert.Error(t, err)
	})
}
