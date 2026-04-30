package grpc

import (
	"context"
	"testing"

	"github.com/jassus213/go-board/dashboard/auth"
	pb "github.com/jassus213/go-board/dashboard/delivery/grpc/gen"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type mockAuth struct{ token string }

func (m *mockAuth) Verify(ctx context.Context, t string) (string, error) {
	if t == m.token {
		return "user1", nil
	}
	return "", auth.ErrInvalidToken
}

func TestAuthInterceptor(t *testing.T) {
	verifier := &mockAuth{token: "valid-pass"}
	interceptor := AuthInterceptor(verifier)
	info := &grpc.StreamServerInfo{FullMethod: "/dashboard.DashboardService/StreamUpdates"}

	t.Run("missing_metadata", func(t *testing.T) {
		err := interceptor(nil, &mockStream{ctx: context.Background()}, info, nil)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Equal(t, "authentication token is required", st.Message())
		assertProblemDetail(t, st.Details(), "auth_missing_token")
	})

	t.Run("valid_token", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer valid-pass")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		handler := func(srv interface{}, stream grpc.ServerStream) error {
			id := stream.Context().Value(memberIDKey)
			assert.Equal(t, "user1", id)
			return nil
		}

		err := interceptor(nil, &mockStream{ctx: ctx}, info, handler)
		assert.NoError(t, err)
	})

	t.Run("missing_authorization_header", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-id", "1"))
		err := interceptor(nil, &mockStream{ctx: ctx}, info, nil)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assertProblemDetail(t, st.Details(), "auth_missing_token")
	})
}

func TestAuthInterceptor_Fail(t *testing.T) {
	verifier := &mockAuth{token: "correct"}
	interceptor := AuthInterceptor(verifier)
	info := &grpc.StreamServerInfo{FullMethod: "/dashboard.DashboardService/StreamUpdates"}

	t.Run("invalid_token", func(t *testing.T) {
		md := metadata.Pairs("authorization", "wrong")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		err := interceptor(nil, &mockStream{ctx: ctx}, info, nil)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
		assertProblemDetail(t, st.Details(), "auth_invalid_token")
	})

	t.Run("handler_error_passthrough", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer correct")
		ctx := metadata.NewIncomingContext(context.Background(), md)

		handler := func(srv interface{}, stream grpc.ServerStream) error {
			return assert.AnError
		}

		err := interceptor(nil, &mockStream{ctx: ctx}, info, handler)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func assertProblemDetail(t *testing.T, details []interface{}, code string) {
	t.Helper()
	if assert.NotEmpty(t, details) {
		pd, ok := details[0].(*pb.ProblemDetails)
		if assert.True(t, ok) {
			assert.Equal(t, code, pd.Code)
		}
	}
}
