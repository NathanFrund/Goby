package websocket

import (
	"log/slog"
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

// SendMessage safely sends a message to the client's send channel.
// It uses a read lock to ensure the channel is not closed concurrently.
func (c *Client) SendMessage(msg []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If the channel is nil, it means the client is disconnected.
	if c.Send == nil {
		return
	}

	select {
	case c.Send <- msg:
	default:
		slog.Warn("Client send channel full, dropping message", "clientID", c.ID)
	}
}

// Close safely closes the client's send channel.
// It uses a write lock to prevent other operations during closing.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the channel is not nil and not already closed
	if c.Send != nil {
		close(c.Send)
		c.Send = nil // Set to nil to prevent further use
	}
}
