package data

import (
	"context"
	"log/slog"

	"github.com/coder/websocket"
	"github.com/nfrund/goby/internal/hub"
)

// Client is a middleman between the WebSocket connection and the data hub.
type Client struct {
	conn       *websocket.Conn
	hub        *hub.Hub
	subscriber *hub.Subscriber
}

// readPump pumps messages from the WebSocket connection to the data hub.
// For this simple data endpoint, we'll just log incoming messages and not broadcast them.
// A real application might process these messages.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister <- c.subscriber
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, msgBytes, err := c.conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				slog.Info("Data WebSocket closed normally")
			} else {
				slog.Error("Data readPump error", "error", err)
			}
			break
		}
		slog.Info("Received message on data channel", "message", string(msgBytes))
	}
}

// writePump pumps messages from the data hub to the WebSocket connection.
func (c *Client) writePump() {
	defer func() {
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()
	for message := range c.subscriber.Send {
		if err := c.conn.Write(context.Background(), websocket.MessageText, message); err != nil {
			slog.Error("Data writePump error", "error", err)
			return
		}
	}
}
