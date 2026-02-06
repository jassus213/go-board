package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	t.Run("register_and_unregister_client", func(t *testing.T) {
		client := &Client{
			hub:  hub,
			send: make(chan OutboundMessage),
		}

		hub.register <- client
		time.Sleep(10 * time.Millisecond)
		assert.True(t, hub.clients[client])

		hub.unregister <- client
		time.Sleep(10 * time.Millisecond)
		assert.False(t, hub.clients[client])
	})
}
