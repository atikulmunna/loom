package server

import (
	"log"
	"net/http"
	"time"

	"github.com/atikulmunna/loom/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// handleWebSocket upgrades to WebSocket and streams log entries to the client.
func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Subscribe to the hub for log entries.
	entries := s.hub.Subscribe()

	// Read pump — detect client disconnect.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				conn.Close()
				return
			}
		}
	}()

	// Write pump — send entries as JSON.
	for entry := range entries {
		msg := struct {
			Timestamp string            `json:"timestamp"`
			Source    string            `json:"source"`
			Level    string            `json:"level"`
			Message  string            `json:"message"`
			Raw      string            `json:"raw"`
			Fields   map[string]string `json:"fields,omitempty"`
		}{
			Timestamp: entry.Timestamp.Format(time.RFC3339),
			Source:    entry.Source,
			Level:    entry.Level,
			Message:  entry.Message,
			Raw:      entry.Raw,
			Fields:   entry.Fields,
		}

		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("websocket write failed: %v", err)
			return
		}
	}
}

// SubscribeForWS is a helper used by the Hub to create a WebSocket-compatible subscriber.
// This is unused—Hub.Subscribe() is called directly in handleWebSocket.
var _ model.LogEntry // ensure model import is used
