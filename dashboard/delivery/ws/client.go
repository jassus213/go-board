package ws

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jassus213/go-board/dashboard/auth"
	"github.com/jassus213/go-board/dashboard/core/dto"
	"github.com/jassus213/go-board/dashboard/core/interfaces"
	"github.com/jassus213/go-board/dashboard/delivery/problem"
)

// Time allowed to write a message to the peer.
const writeWait = 10 * time.Second

// Time allowed to read the next pong message from the peer.
const pongWait = 60 * time.Second

// Send pings to peer with this period. Must be less than pongWait.
const pingPeriod = (pongWait * 9) / 10

// CORSConfig holds CORS configuration for WebSocket connections.
type CORSConfig struct {
	// AllowedOrigins is a list of allowed origins for WebSocket connections.
	// Use ["*"] to allow all origins (NOT recommended for production).
	AllowedOrigins []string
	// AllowCredentials indicates whether credentials are allowed.
	AllowCredentials bool
}

// upgrader handles the WebSocket handshake and protocol upgrade.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin will be set dynamically per request
	CheckOrigin: nil,
}

// Client represents a single active WebSocket connection.
// It acts as a middleman between the websocket connection and the Hub,
// handling the specific business logic for dashboard updates.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	uc       interfaces.DashboardUseCase
	send     chan OutboundMessage
	memberID string
}

// readPump pumps messages from the websocket connection to the hub/application.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
//
// Workflow:
// 1. Reads JSON (InboundMessage) from the client.
// 2. Executes the "Increment Score" command via the BLL.
// 3. Executes the "Get Rank" query via the BLL.
// 4. Sends the result back to the client's write channel.
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	// Set configuration for connection health checks (Pong)
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg InboundMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			break
		}

		targetID := c.memberID
		if msg.MemberID != "" && msg.MemberID != c.memberID {
			log.Printf("User %s tried to update score for %s. Overriding to %s", c.memberID, msg.MemberID, c.memberID)
		}

		rank, err := c.uc.ProcessScoreUpdate(ctx, dto.IncrementScoreRequest{
			Dashboard: msg.Dashboard,
			MemberID:  targetID,
			Value:     msg.Increment,
		})

		response := OutboundMessage{}
		if err != nil {
			pd := problem.FromError(err, http.StatusInternalServerError, "/ws")
			response.Problem = &pd
		} else {
			response.Rank = rank
		}

		c.send <- response
	}
}

// writePump pumps messages from the hub/application to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
// It also handles the periodic sending of Ping messages to keep the connection alive.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.writeMessage(message, ok); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.sendPing(); err != nil {
				return
			}
		}
	}
}

func (c *Client) writeMessage(message OutboundMessage, ok bool) error {
	if err := c.setWriteDeadline(); err != nil {
		return err
	}

	if !ok {
		return c.conn.WriteMessage(websocket.CloseMessage, []byte{})
	}

	return c.conn.WriteJSON(message)
}

func (c *Client) sendPing() error {
	if err := c.setWriteDeadline(); err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

func (c *Client) setWriteDeadline() error {
	return c.conn.SetWriteDeadline(time.Now().Add(writeWait))
}

// ServeWs handles websocket requests from the peer.
// It upgrades the HTTP connection to a WebSocket connection, creates a new Client,
// registers it with the Hub, and starts the read/write pumps.
func ServeWs(
	hub *Hub,
	uc interfaces.DashboardUseCase,
	verifier auth.Verifier,
	corsConfig CORSConfig,
	w http.ResponseWriter,
	r *http.Request,
) {
	// Set CheckOrigin dynamically based on CORS config
	upgrader.CheckOrigin = createOriginChecker(corsConfig)

	token := r.URL.Query().Get("token")

	if token == "" {
		token = r.Header.Get("Sec-WebSocket-Protocol")
	}

	if token == "" {
		pd := problem.FromError(auth.ErrEmptyToken, http.StatusUnauthorized, r.URL.Path)
		problem.WriteHTTP(w, &pd)
		return
	}

	memberID, err := verifier.Verify(r.Context(), token)
	if err != nil {
		log.Printf("WebSocket Auth failed: %v", err)
		pd := problem.FromError(err, http.StatusForbidden, r.URL.Path)
		problem.WriteHTTP(w, &pd)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		uc:       uc,
		send:     make(chan OutboundMessage, 256),
		memberID: memberID,
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump(r.Context())
}

// createOriginChecker creates a CheckOrigin function based on CORS configuration.
func createOriginChecker(config CORSConfig) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		// If no origin header, allow (same-origin request)
		if origin == "" {
			return true
		}

		// Check if origin is in allowed list
		for _, allowed := range config.AllowedOrigins {
			if allowed == "*" {
				// Wildcard - allow all (not recommended for production)
				log.Printf("WARNING: CORS wildcard (*) is enabled. This is insecure for production!")
				return true
			}
			if origin == allowed {
				return true
			}
		}

		// Origin not allowed.
		log.Printf("WebSocket connection rejected: origin is not allowed")
		return false
	}
}
