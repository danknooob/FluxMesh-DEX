package server

import (
	"log"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/notification/internal/hub"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In dev, accept all origins. In prod, restrict this.
		return true
	},
}

// WSHandler upgrades HTTP connections to WebSockets and wires them into the hub.
// For now, user_id is taken from a query param; later it should come from JWT.
func WSHandler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "missing user_id", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws: upgrade error: %v", err)
			return
		}

		client := &hub.Client{
			UserID: userID,
			Send:   make(chan []byte, 256),
		}
		h.Register(client)

		// Writer goroutine.
		go func() {
			defer func() {
				h.Unregister(client)
				conn.Close()
			}()
			for msg := range client.Send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Printf("ws: write error: %v", err)
					return
				}
			}
		}()

		// Reader loop (no messages expected yet; just detect close).
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}
}

