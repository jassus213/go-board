package ws

// Hub maintains the set of active clients and handles thread-safe registration
// and unregistration requests via channels.
//
// It acts as a central manager that ensures the map of clients is only modified
// by a single goroutine (the Run loop), preventing race conditions.
type Hub struct {
	// Registered clients.
	// We use a map for fast O(1) lookups and deletions.
	clients map[*Client]bool

	// Inbound channel for registering new clients.
	register chan *Client

	// Inbound channel for unregistering clients.
	// When a client disconnects, it sends itself here to be removed from the map.
	unregister chan *Client
	broadcast  chan OutboundMessage
}

// NewHub creates a new Hub instance with initialized channels and client map.
func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		broadcast:  make(chan OutboundMessage),
	}
}

// Run starts the main event loop of the Hub.
// It continuously listens for register and unregister events.
// This method blocks and should typically be executed in a separate goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			h.unregisterClient(client)
		case msg := <-h.broadcast:
			h.broadcastMessage(msg)
		}
	}
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	select {
	case <-client.send:
	default:
		close(client.send)
	}
}

func (h *Hub) broadcastMessage(msg OutboundMessage) {
	for client := range h.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}
