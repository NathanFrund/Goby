package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/topics"
	wsTopics "github.com/nfrund/goby/internal/topics/websocket"
)

type ConnectionType int

const (
	// ConnectionTypeHTML is for clients that consume HTML fragments (e.g., HTMX).
	ConnectionTypeHTML ConnectionType = iota
	// ConnectionTypeData is for clients that consume structured data (e.g., JSON).
	ConnectionTypeData
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// Bridge handles WebSocket connections for a specific endpoint ("html" or "data").
type Bridge struct {
	endpoint      string
	publisher     pubsub.Publisher
	subscriber    pubsub.Subscriber
	topicRegistry *topics.TopicRegistry
	readyTopic    topics.Topic
	clients       *ClientManager
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// BridgeDependencies contains all dependencies required by the Bridge.
type BridgeDependencies struct {
	Publisher     pubsub.Publisher
	Subscriber    pubsub.Subscriber
	TopicRegistry *topics.TopicRegistry
	ReadyTopic    topics.Topic
}

// NewBridge creates a new WebSocket bridge for a specific endpoint.
func NewBridge(endpoint string, deps BridgeDependencies) *Bridge {
	return &Bridge{
		endpoint:      endpoint,
		publisher:     deps.Publisher,
		subscriber:    deps.Subscriber,
		topicRegistry: deps.TopicRegistry,
		readyTopic:    deps.ReadyTopic,
		clients:       NewClientManager(),
	}
}

// Start begins the bridge's message handling loop, subscribing to relevant pub/sub topics.
// Returns an error if any subscription fails.
func (b *Bridge) Start(ctx context.Context) error {
	// Create a cancellable context for this bridge
	var bridgeCtx context.Context
	bridgeCtx, b.cancel = context.WithCancel(ctx)

	// Register WebSocket topics
	wsTopics.MustRegisterAll(b.topicRegistry)

	// Determine which topics to use based on the endpoint
	var broadcastTopic, directTopic topics.Topic

	switch b.endpoint {
	case "html":
		broadcastTopic = wsTopics.HTMLBroadcast
		directTopic = wsTopics.HTMLDirect
	case "data":
		broadcastTopic = wsTopics.DataBroadcast
		directTopic = wsTopics.DataDirect
	default:
		err := fmt.Errorf("unknown endpoint type: %s", b.endpoint)
		slog.Error("FATAL: Invalid endpoint type", "endpoint", b.endpoint, "error", err)
		return err
	}

	// Subscribe to broadcast messages for this endpoint
	if err := b.subscriber.Subscribe(bridgeCtx, broadcastTopic.Pattern(), b.handleBroadcast); err != nil {
		slog.Error("FATAL: Failed to subscribe to broadcast topic",
			"topic", broadcastTopic.Name(),
			"pattern", broadcastTopic.Pattern(),
			"error", err)
		return fmt.Errorf("failed to subscribe to broadcast topic %s: %w", broadcastTopic.Name(), err)
	}

	// Subscribe to direct messages for this endpoint using the pattern
	// The pattern will match any direct message for this endpoint (e.g., "ws.html.direct.*")
	if err := b.subscriber.Subscribe(bridgeCtx, directTopic.Pattern(), b.handleDirectMessage); err != nil {
		slog.Error("FATAL: Failed to subscribe to direct message topic",
			"topic", directTopic.Name(),
			"pattern", directTopic.Pattern(),
			"error", err)
		return fmt.Errorf("failed to subscribe to direct message topic %s: %w", directTopic.Name(), err)
	}

	slog.Info("Successfully subscribed to WebSocket topics",
		"endpoint", b.endpoint,
		"broadcast_topic", broadcastTopic.Pattern(),
		"direct_topic_pattern", directTopic.Pattern())

	return nil
}

func (b *Bridge) handleBroadcast(ctx context.Context, msg pubsub.Message) error {
	clients := b.clients.GetAll()
	for _, client := range clients {
		// SendMessage handles its own error logging
		client.SendMessage(msg.Payload)
	}
	return nil
}

// handleDirectMessage processes direct messages for specific clients
// The recipient ID should be specified in the message metadata as "recipient_id"
func (b *Bridge) handleDirectMessage(ctx context.Context, msg pubsub.Message) error {
	// Get recipient ID from metadata
	recipientID, exists := msg.Metadata["recipient_id"]
	if !exists || recipientID == "" {
		slog.Warn("Direct message missing recipient_id in metadata", "metadata", msg.Metadata)
		return nil
	}

	// Get all active clients for this recipient
	clients := b.clients.GetByUser(recipientID)
	if len(clients) == 0 {
		slog.Debug("No active clients found for recipient", "recipient", recipientID, "endpoint", b.endpoint)
		return nil
	}

	// Forward the message to all of the recipient's clients for this endpoint
	for _, client := range clients {
		if client.Endpoint == b.endpoint {
			client.SendMessage(msg.Payload)
		}
	}

	return nil
}

// Handler returns an echo.HandlerFunc that handles WebSocket upgrade requests for a given connection type.
// Shutdown gracefully stops the bridge's background processes.
// The provided context is used for the shutdown timeout.
func (b *Bridge) Shutdown(ctx context.Context) {
	slog.Info("Shutting down WebSocket bridge", "endpoint", b.endpoint)
	if b.cancel != nil {
		b.cancel() // This will cause the pub/sub subscriptions to terminate
	}

	// Create a channel to signal when shutdown is complete
	done := make(chan struct{})
	go func() {
		// Wait for all read/write pumps to finish
		b.wg.Wait()
		close(done)
	}()

	// Wait for either shutdown to complete or context to be cancelled
	select {
	case <-done:
		slog.Info("WebSocket bridge shut down gracefully", "endpoint", b.endpoint)
	case <-ctx.Done():
		slog.Warn("WebSocket bridge shutdown timed out", "endpoint", b.endpoint, "error", ctx.Err())
	}
}

func (b *Bridge) Handler() echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := c.Get(middleware.UserContextKey).(*domain.User)
		if !ok || user == nil {
			slog.Error("Bridge.serve: Could not get user from context for WebSocket connection")
			return c.String(http.StatusUnauthorized, "User not authenticated")
		}

		conn, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
			// In production, you should verify the origin of the request against a list of
			// allowed origins to prevent cross-site WebSocket hijacking.
			InsecureSkipVerify: true, // TODO: Replace with a proper origin check in production.
		})
		if err != nil {
			slog.Error("Failed to upgrade connection to WebSocket", "error", err, "userID", user.Email)
			return fmt.Errorf("failed to upgrade connection to WebSocket: %w", err)
		}

		client := &Client{
			ID:       watermill.NewUUID(),
			UserID:   user.Email,
			Conn:     conn,
			Send:     make(chan []byte, 256),
			Endpoint: b.endpoint,
		}

		// Register the client
		b.clients.Add(client)

		// Publish a "ready" event to the message bus so other modules can react.
		// This is done in a goroutine to avoid blocking the connection handler.
		go func() {
			payload, _ := json.Marshal(map[string]any{
				"userID":   client.UserID,
				"endpoint": client.Endpoint,
			})
			readyMsg := pubsub.Message{
				Topic:   b.readyTopic.Name(),
				UserID:  client.UserID,
				Payload: payload,
			}
			if err := b.publisher.Publish(context.Background(), readyMsg); err != nil {
				slog.Error("Failed to publish websocket ready event", "error", err, "userID", client.UserID)
			}
		}()

		// Start the read and write pumps
		b.wg.Add(2)
		go b.writePump(client)
		go b.readPump(client)

		return nil
	}
}

// readPump pumps messages from the WebSocket connection to the bridge's incoming channel.
func (b *Bridge) readPump(client *Client) {
	defer func() {
		b.clients.Remove(client.ID)
		b.wg.Done()
		slog.Info("Client disconnected", "clientID", client.ID, "userID", client.UserID, "endpoint", b.endpoint)
	}()

	// The coder/websocket library does not have SetReadLimit, so we check manually.
	// It does, however, automatically handle pong messages to update the read deadline.
	for {
		_, message, err := client.Conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				slog.Debug("WebSocket closed normally by client", "clientID", client.ID)
			} else {
				slog.Error("Unexpected WebSocket read error", "clientID", client.ID, "error", err)
			}
			break
		}

		// Check message size since we can't set a read limit directly
		if len(message) > maxMessageSize {
			slog.Warn("Message too large, closing connection",
				"clientID", client.ID,
				"size", len(message),
				"max", maxMessageSize)
			return
		}

		// Standardize incoming messages. They must be JSON with an "action" and "payload".
		var inboundMsg struct {
			Action  string          `json:"action"`
			Payload json.RawMessage `json:"payload"`
		}

		if err := json.Unmarshal(message, &inboundMsg); err != nil {
			slog.Warn("Received invalid message from client", "clientID", client.ID, "error", err)
			continue // Ignore malformed messages
		}

		if inboundMsg.Action == "" {
			slog.Warn("Incoming message missing 'action' field", "clientID", client.ID)
			continue
		}

		// Use the message's "action" as the pub/sub topic.
		b.publisher.Publish(context.Background(), pubsub.Message{
			Topic:   inboundMsg.Action,
			Payload: inboundMsg.Payload,
			UserID:  client.UserID,
		})
	}
}

func (b *Bridge) writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close(websocket.StatusNormalClosure, "write pump closing")
		b.wg.Done()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// The manager closed the channel.
				client.Conn.Close(websocket.StatusNormalClosure, "channel closed")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := client.Conn.Write(ctx, websocket.MessageText, message)
			cancel()
			if err != nil {
				slog.Warn("WebSocket write error", "clientID", client.ID, "error", err)
				return
			}

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := client.Conn.Ping(ctx)
			cancel()
			if err != nil {
				slog.Warn("WebSocket ping error", "clientID", client.ID, "error", err)
				return
			}
		}
	}
}
