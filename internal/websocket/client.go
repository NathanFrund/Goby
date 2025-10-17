package websocket

import (
	"sync"

	"github.com/coder/websocket"
)

// Client represents a single connected WebSocket client.
type Client struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	Endpoint string // "html" or "data"
	mu       sync.RWMutex
}
