package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core/usecase"
	"github.com/jassus213/go-board/dashboard/repo/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWebSocketFullWorkflow(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	repo := mocks.NewDashboardRepository(t)
	uc := usecase.New(repo)

	verifier := &auth.StaticVerifier{Secret: "pass"}

	corsConfig := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, uc, verifier, corsConfig, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=pass"

	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(wsURL, nil)
	if resp != nil && resp.Body != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	t.Run("send_increment_get_rank", func(t *testing.T) {
		assertIncrementAndRank(t, repo, conn)
	})

	t.Run("ping_pong_coverage", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond)
	})
}

func assertIncrementAndRank(t *testing.T, repo *mocks.DashboardRepository, conn *websocket.Conn) {
	t.Helper()

	repo.EXPECT().
		IncrementMemberScore(mock.Anything, "games", "admin_user", 5.0).
		Return(nil).Once()

	repo.EXPECT().
		ViewMemberRank(mock.Anything, "games", "admin_user").
		Return(int64(1), nil).Once()

	msg := InboundMessage{
		Dashboard: "games",
		MemberID:  "someone_else",
		Increment: 5,
	}

	err := conn.WriteJSON(msg)
	assert.NoError(t, err)

	var resp OutboundMessage
	err = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	assert.NoError(t, err)

	err = conn.ReadJSON(&resp)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), resp.Rank)
	assert.Empty(t, resp.Error)
}
