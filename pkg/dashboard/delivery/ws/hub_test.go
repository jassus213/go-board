package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.register)

	t.Run("register_and_unregister_client", func(t *testing.T) {
		client := &Client{
			hub:  hub,
			send: make(chan OutboundMessage, 1),
		}

		hub.register <- client
		time.Sleep(50 * time.Millisecond)

		done := make(chan bool, 1)
		go func() {
			hub.broadcast <- OutboundMessage{Rank: 1}
			done <- true
		}()

		select {
		case <-done:
			// Broadcast succeeded, client is registered
			assert.True(t, true)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Broadcast blocked, client may not be registered")
		}

		select {
		case <-client.send:
		case <-time.After(50 * time.Millisecond):
		}

		hub.unregister <- client
		time.Sleep(50 * time.Millisecond)

		select {
		case _, ok := <-client.send:
			assert.False(t, ok, "client channel should be closed after unregister")
		case <-time.After(50 * time.Millisecond):
			t.Fatal("Expected client channel to be closed")
		}
	})
}
