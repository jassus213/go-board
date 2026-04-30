package ws

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core/usecase"
	"github.com/jassus213/go-board/dashboard/repo/mocks"
	"github.com/stretchr/testify/assert"
)

type mockVerifier struct{}

func (m *mockVerifier) Verify(ctx context.Context, t string) (string, error) {
	if t == "valid" {
		return "u1", nil
	}
	return "", auth.ErrInvalidToken
}

func TestServeWs_Auth(t *testing.T) {
	hub := NewHub()
	repo := mocks.NewDashboardRepository(t)
	uc := usecase.New(repo)

	verifier := &mockVerifier{}

	corsConfig := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	}

	t.Run("unauthorized_missing_token", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/ws", nil)
		w := httptest.NewRecorder()

		ServeWs(hub, uc, verifier, corsConfig, w, req)

		assert.Equal(t, 401, w.Code)
		assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
		var resp map[string]interface{}
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "auth_missing_token", resp["code"])
	})

	t.Run("forbidden_invalid_token", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/ws?token=wrong", nil)
		w := httptest.NewRecorder()

		ServeWs(hub, uc, verifier, corsConfig, w, req)

		assert.Equal(t, 403, w.Code)
		assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
		var resp map[string]interface{}
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "auth_invalid_token", resp["code"])
	})
}
