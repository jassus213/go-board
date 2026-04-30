package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core/entity"
	"github.com/jassus213/go-board/dashboard/core/usecase"
	"github.com/jassus213/go-board/dashboard/repo/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRouter_HealthAndAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := mocks.NewDashboardRepository(t)
	router := NewRouter(&Params{
		UseCase:  usecase.New(repo),
		Verifier: &auth.StaticVerifier{Secret: "token"},
		Config:   Config{},
	})

	t.Run("health_ok", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/health", http.NoBody)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
	})

	t.Run("rest_unauthorized", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/dashboards/games/stats", http.NoBody)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "auth_missing_token", body["code"])
	})

	t.Run("rest_forbidden_with_invalid_token", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/dashboards/games/stats", http.NoBody)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer wrong-token")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "auth_invalid_token", body["code"])
	})

	t.Run("websocket_disabled_endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})
}

func TestRouter_LeaderboardHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := mocks.NewDashboardRepository(t)
	router := NewRouter(&Params{
		UseCase:  usecase.New(repo),
		Verifier: &auth.StaticVerifier{Secret: "token"},
		Config:   Config{},
	})

	t.Run("increment_score_success", func(t *testing.T) {
		repo.EXPECT().
			IncrementMemberScore(mock.Anything, "games", "admin_user", 5.0).
			Return(nil).
			Once()
		repo.EXPECT().
			ViewMemberRank(mock.Anything, "games", "admin_user").
			Return(int64(2), nil).
			Once()

		payload := []byte(`{"increment":5}`)
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"/api/v1/dashboards/games/members/u1/increment",
			bytes.NewReader(payload),
		)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "admin_user", body["member_id"])
		assert.EqualValues(t, 2, body["rank"])
	})

	t.Run("increment_score_invalid_payload", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"/api/v1/dashboards/games/members/u1/increment",
			bytes.NewReader([]byte(`{}`)),
		)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("top_members_invalid_limit", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/api/v1/dashboards/games/top?limit=bad",
			http.NoBody,
		)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "invalid_argument", body["code"])
	})

	t.Run("top_members_success", func(t *testing.T) {
		repo.EXPECT().
			GetTopMembers(mock.Anything, "games", int64(3)).
			Return([]entity.DashboardRecord{
				{ID: "u1", Rank: 1, Score: 100},
				{ID: "u2", Rank: 2, Score: 90},
			}, nil).
			Once()

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/api/v1/dashboards/games/top?limit=3",
			http.NoBody,
		)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Members []entity.DashboardRecord `json:"members"`
		}
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Len(t, body.Members, 2)
		assert.Equal(t, "u1", body.Members[0].ID)
	})

	t.Run("member_rank_success", func(t *testing.T) {
		repo.EXPECT().
			ViewMemberRank(mock.Anything, "games", "admin_user").
			Return(int64(7), nil).
			Once()

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/api/v1/dashboards/games/members/u2/rank",
			http.NoBody,
		)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "admin_user", body["member_id"])
		assert.EqualValues(t, 7, body["rank"])
	})

	t.Run("dashboard_stats_success", func(t *testing.T) {
		repo.EXPECT().
			GetTotalMembers(mock.Anything, "games").
			Return(int64(42), nil).
			Once()

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/api/v1/dashboards/games/stats",
			http.NoBody,
		)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.EqualValues(t, 42, body["total_members"])
	})

	t.Run("dashboard_stats_usecase_error", func(t *testing.T) {
		repo.EXPECT().
			GetTotalMembers(mock.Anything, "games").
			Return(int64(0), assert.AnError).
			Once()

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodGet,
			"/api/v1/dashboards/games/stats",
			http.NoBody,
		)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
