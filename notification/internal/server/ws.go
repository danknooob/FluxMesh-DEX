package server

import (
	"log"
	"net/http"

	"github.com/danknooob/fluxmesh-dex/notification/internal/hub"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// WSHandler upgrades HTTP connections to WebSockets after validating the
// JWT token supplied as a ?token= query parameter. The browser WebSocket
// API cannot send custom headers, so query-param auth is the standard
// approach for WS connections.
func WSHandler(h *hub.Hub, jwtSecret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("token")
		if raw == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(raw, &claims{}, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		c, ok := token.Claims.(*claims)
		if !ok || !token.Valid || c.UserID == "" {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws: upgrade error: %v", err)
			return
		}

		client := &hub.Client{
			UserID: c.UserID,
			Send:   make(chan []byte, 256),
		}
		h.Register(client)

		go func() {
			defer func() {
				h.Unregister(client)
				_ = conn.Close()
			}()
			for msg := range client.Send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Printf("ws: write error: %v", err)
					return
				}
			}
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}
}
