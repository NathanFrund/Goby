package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/hub"
)

// Client is a middleman between the WebSocket connection and the hub.
type Client struct {
	// The WebSocket connection.
	conn *websocket.Conn

	// The hub to which the client is registered.
	hub *hub.Hub

	// The subscriber instance for this client, containing the outbound message channel.
	subscriber *hub.Subscriber

	// The authenticated user associated with this client.
	User *domain.User

	// A reference to the template renderer.
	renderer echo.Renderer
}

// readPump pumps messages from the WebSocket connection to the hub.
//
// The application runs one readPump per connection. It ensures that there is at
// most one reader on a connection by executing all reads from this goroutine.
func (c *Client) readPump() {
	// When this function returns, it means the client has disconnected.
	// We unregister the client and close the connection.
	defer func() {
		c.hub.Unregister <- c.subscriber
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	// Optional: Set a read limit on messages to prevent abuse.
	// c.conn.SetReadLimit(512)

	for {
		// Read a message from the WebSocket.
		slog.Debug("readPump: Waiting for message from client")
		_, msgBytes, err := c.conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				slog.Info("WebSocket closed normally")
			} else {
				slog.Error("readPump error", "error", err)
			}
			break // Exit the loop to trigger the defer statement.
		}

		slog.Info("readPump: Received message from client", "raw_message", string(msgBytes))
		// We expect incoming messages to be simple JSON with just content.
		var incoming struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(msgBytes, &incoming); err != nil {
			slog.Error("Error unmarshalling incoming chat message", "error", err)
			continue
		}

		// Use the user stored on the client.
		if c.User == nil {
			slog.Error("WebSocket client has no associated user")
			continue
		}

		// Determine the username. Use Name if available, otherwise Email.
		var username string
		if c.User.Name != nil && *c.User.Name != "" {
			username = *c.User.Name
		} else {
			username = c.User.Email
		}

		// Create a full Message object and broadcast it to the hub.
		chatMessage := Message{
			UserID:   c.User.ID.String(),
			Username: username,
			Content:  incoming.Content,
			SentAt:   time.Now(),
		}

		// --- Render the message to an HTML fragment ---
		var buf bytes.Buffer
		err = c.renderer.Render(&buf, "chat-message.html", chatMessage, nil)
		if err != nil {
			slog.Error("readPump: Error rendering chat message template", "error", err)
			continue
		}

		renderedHTML := buf.Bytes()

		// --- Broadcast the rendered HTML fragment to the hub ---
		slog.Info("readPump: Broadcasting rendered HTML to hub", "html_bytes", len(renderedHTML))
		c.hub.Broadcast <- renderedHTML
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by

func (c *Client) writePump() {
	defer func() {
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for message := range c.subscriber.Send {
		slog.Debug("writePump: Received message from hub to send to client")

		// The message is already a pre-rendered HTML fragment ([]byte).
		slog.Info("writePump: Writing HTML fragment to WebSocket", "html_bytes", len(message))
		// Write the HTML fragment to the WebSocket.
		if err := c.conn.Write(context.Background(), websocket.MessageText, message); err != nil {
			slog.Error("writePump error", "error", err)
			return
		}
	}
}
