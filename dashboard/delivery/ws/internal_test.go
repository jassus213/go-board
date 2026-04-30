package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOriginChecker(t *testing.T) {
	t.Run("allows empty origin", func(t *testing.T) {
		check := createOriginChecker(CORSConfig{AllowedOrigins: []string{"https://allowed.com"}})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
		assert.True(t, check(req))
	})

	t.Run("allows exact match", func(t *testing.T) {
		check := createOriginChecker(CORSConfig{AllowedOrigins: []string{"https://allowed.com"}})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
		req.Header.Set("Origin", "https://allowed.com")
		assert.True(t, check(req))
	})

	t.Run("allows wildcard", func(t *testing.T) {
		check := createOriginChecker(CORSConfig{AllowedOrigins: []string{"*"}})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
		req.Header.Set("Origin", "https://any.com")
		assert.True(t, check(req))
	})

	t.Run("rejects unknown origin", func(t *testing.T) {
		check := createOriginChecker(CORSConfig{AllowedOrigins: []string{"https://allowed.com"}})
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ws", http.NoBody)
		req.Header.Set("Origin", "https://blocked.com")
		assert.False(t, check(req))
	})
}

func TestHubUnregisterClient(t *testing.T) {
	t.Run("missing client does nothing", func(t *testing.T) {
		h := NewHub()
		client := &Client{send: make(chan OutboundMessage, 1)}
		h.unregisterClient(client)
		assert.Empty(t, h.clients)
	})

	t.Run("client with empty channel is closed", func(t *testing.T) {
		h := NewHub()
		client := &Client{send: make(chan OutboundMessage, 1)}
		h.clients[client] = true

		h.unregisterClient(client)
		assert.NotContains(t, h.clients, client)

		_, ok := <-client.send
		assert.False(t, ok)
	})

	t.Run("client with queued message keeps channel open", func(t *testing.T) {
		h := NewHub()
		client := &Client{send: make(chan OutboundMessage, 1)}
		client.send <- OutboundMessage{Rank: 1}
		h.clients[client] = true

		h.unregisterClient(client)
		assert.NotContains(t, h.clients, client)

		assert.NotPanics(t, func() {
			close(client.send)
		})
	})
}

func TestHubBroadcastMessage(t *testing.T) {
	h := NewHub()
	good := &Client{send: make(chan OutboundMessage, 1)}
	blocked := &Client{send: make(chan OutboundMessage, 1)}
	blocked.send <- OutboundMessage{Rank: 99}

	h.clients[good] = true
	h.clients[blocked] = true

	msg := OutboundMessage{Rank: 7}
	h.broadcastMessage(msg)

	select {
	case got := <-good.send:
		assert.Equal(t, msg, got)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected broadcast message for active client")
	}

	_, ok := <-blocked.send
	assert.True(t, ok) // existing queued message
	_, ok = <-blocked.send
	assert.False(t, ok) // closed by hub
	assert.NotContains(t, h.clients, blocked)
}

func TestClientSendPingAndWritePumpShutdown(t *testing.T) {
	serverConnCh := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConnCh <- conn
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	defer func() { _ = clientConn.Close() }()

	serverConn := <-serverConnCh
	defer func() { _ = serverConn.Close() }()

	c := &Client{
		conn: clientConn,
		send: make(chan OutboundMessage, 1),
	}

	require.NoError(t, c.sendPing())

	done := make(chan struct{})
	go func() {
		c.writePump()
		close(done)
	}()

	close(c.send)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("writePump did not exit after send channel close")
	}
}
