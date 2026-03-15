package hub

import (
	"log"
	"sync"
)

// Message represents a notification destined for a specific user.
type Message struct {
	UserID string
	Data   []byte
}

// Client represents a WebSocket connection subscribed for a given user.
type Client struct {
	UserID string
	Send   chan []byte
}

// Hub maintains active clients and broadcasts messages to them.
type Hub struct {
	mu          sync.RWMutex
	clients     map[string]map[*Client]struct{} // userId -> set of clients
	register    chan *Client
	unregister  chan *Client
	broadcast   chan Message
	broadcastAll chan []byte
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:     make(map[string]map[*Client]struct{}),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan Message, 1024),
		broadcastAll: make(chan []byte, 256),
	}
}

// Run starts the hub event loop.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			set, ok := h.clients[c.UserID]
			if !ok {
				set = make(map[*Client]struct{})
				h.clients[c.UserID] = set
			}
			set[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if set, ok := h.clients[c.UserID]; ok {
				delete(set, c)
				if len(set) == 0 {
					delete(h.clients, c.UserID)
				}
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			set := h.clients[msg.UserID]
			h.mu.RUnlock()
			for client := range set {
				select {
				case client.Send <- msg.Data:
				default:
					log.Printf("notification: dropping message for user %s (slow client)", msg.UserID)
				}
			}
		case data := <-h.broadcastAll:
			h.mu.RLock()
			for _, set := range h.clients {
				for client := range set {
					select {
					case client.Send <- data:
					default:
						log.Printf("notification: dropping broadcast (slow client)")
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register enqueues a client registration.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister enqueues a client removal.
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

// Broadcast enqueues a message to all clients for a given user.
func (h *Hub) Broadcast(userID string, data []byte) {
	h.broadcast <- Message{UserID: userID, Data: data}
}

// BroadcastAll enqueues a message to every connected client (e.g. depth_updated).
func (h *Hub) BroadcastAll(data []byte) {
	select {
	case h.broadcastAll <- data:
	default:
		log.Printf("notification: broadcastAll channel full, dropping depth update")
	}
}

