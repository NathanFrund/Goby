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
	"github.com/nfrund/goby/internal/topicmgr"
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
	endpoint     string
	publisher    pubsub.Publisher
	subscriber   pubsub.Subscriber
	topicManager *topicmgr.Manager
	readyTopic   topicmgr.Topic
	clients      *ClientManager
	topics       *topicManager
	whitelist    *clientWhitelist
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// BridgeDependencies contains all dependencies required by the Bridge.
type BridgeDependencies struct {
	Publisher    pubsub.Publisher
	Subscriber   pubsub.Subscriber
	TopicManager *topicmgr.Manager
	ReadyTopic   topicmgr.Topic
}

// topicManager manages topic subscriptions for clients
type topicManager struct {
	sync.RWMutex
	subscriptions map[string]map[string]struct{} // topic -> clientID -> struct{}
}

func newTopicManager() *topicManager {
	return &topicManager{
		subscriptions: make(map[string]map[string]struct{}),
	}
}

// Subscribe adds a client to a topic
type SubscribeMessage struct {
	Action  string `json:"action"` // "subscribe" or "unsubscribe"
	Topic   string `json:"topic"`  // The topic to subscribe to
	Payload struct {
		Channel string `json:"channel,omitempty"` // Optional channel name
	} `json:"payload,omitempty"`
}

// NewBridge creates a new WebSocket bridge for a specific endpoint.
func NewBridge(endpoint string, deps BridgeDependencies) *Bridge {
	return &Bridge{
		endpoint:     endpoint,
		publisher:    deps.Publisher,
		subscriber:   deps.Subscriber,
		topicManager: deps.TopicManager,
		readyTopic:   deps.ReadyTopic,
		clients:      NewClientManager(),
		topics:       newTopicManager(),
		whitelist:    DefaultClientWhitelist(),
	}
}

// AllowAction adds an action to the whitelist of allowed client actions.
// This can be used by modules to register their allowed actions during initialization.
// Returns an error if the action is invalid or already exists.
func (b *Bridge) AllowAction(action string) error {
	if b.whitelist == nil {
		b.whitelist = NewClientWhitelist()
	}

	err := b.whitelist.AddAction(action)
	if err != nil && err != ErrActionAlreadyExists {
		slog.Error("Failed to add action to whitelist",
			"action", action,
			"error", err)
		return err
	}

	return nil
}

// Start begins the bridge's message handling loop, subscribing to relevant pub/sub topics.
// Returns an error if any subscription fails.
func (b *Bridge) Start(ctx context.Context) error {
	// Create a cancellable context for this bridge
	var bridgeCtx context.Context
	bridgeCtx, b.cancel = context.WithCancel(ctx)

	// Get the topics for this endpoint
	broadcastTopic, directTopic, err := b.getEndpointTopics()
	if err != nil {
		return fmt.Errorf("failed to get endpoint topics: %w", err)
	}

	// Subscribe to broadcast messages for this endpoint
	if err := b.subscriber.Subscribe(bridgeCtx, broadcastTopic.Name(), b.handleBroadcast); err != nil {
		slog.Error("FATAL: Failed to subscribe to broadcast topic",
			"topic", broadcastTopic.Name(),
			"error", err)
		return fmt.Errorf("failed to subscribe to broadcast topic %s: %w", broadcastTopic.Name(), err)
	}

	// Subscribe to the direct messages topic
	if err := b.subscriber.Subscribe(bridgeCtx, directTopic.Name(), b.handleDirectMessage); err != nil {
		slog.Error("FATAL: Failed to subscribe to direct topic",
			"topic", directTopic.Name(),
			"error", err)
		return fmt.Errorf("failed to subscribe to direct topic %s: %w", directTopic.Name(), err)
	}

	slog.Info("Successfully subscribed to WebSocket topics",
		"endpoint", b.endpoint,
		"broadcast_topic", broadcastTopic.Name(),
		"direct_topic", directTopic.Name())

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
		slog.Warn("Direct message missing recipient_id in metadata",
			"topic", msg.Topic,
			"metadata", msg.Metadata,
		)
		return nil
	}

	// Get all active clients for this recipient
	clients := b.clients.GetByUser(recipientID)
	if len(clients) == 0 {
		slog.Debug("No active clients found for recipient",
			"recipient", recipientID,
			"endpoint", b.endpoint,
		)
		return nil
	}

	// Forward the message to all of the recipient's clients for this endpoint
	var sentTo int
	for _, client := range clients {
		if client.Endpoint == b.endpoint {
			client.SendMessage(msg.Payload)
			sentTo++
		}
	}

	if sentTo == 0 {
		slog.Debug("No active clients received the direct message",
			"recipientID", recipientID,
			"endpoint", b.endpoint,
		)
	}

	return nil
}

// getEndpointTopics returns the broadcast and direct topics for the bridge's endpoint.
// It looks up the topics from the topic manager using well-known topic names.
func (b *Bridge) getEndpointTopics() (topicmgr.Topic, topicmgr.Topic, error) {
	var broadcastTopic, directTopic topicmgr.Topic

	switch b.endpoint {
	case "html":
		broadcastTopic = TopicHTMLBroadcast
		directTopic = TopicHTMLDirect
	case "data":
		broadcastTopic = TopicDataBroadcast
		directTopic = TopicDataDirect
	default:
		return nil, nil, fmt.Errorf("invalid endpoint: %s", b.endpoint)
	}

	// Verify topics are registered
	if !b.topicManager.CheckTopicExists(broadcastTopic.Name()) {
		return nil, nil, fmt.Errorf("broadcast topic not registered: %s", broadcastTopic.Name())
	}

	if !b.topicManager.CheckTopicExists(directTopic.Name()) {
		return nil, nil, fmt.Errorf("direct topic not registered: %s", directTopic.Name())
	}

	return broadcastTopic, directTopic, nil
}

// Handler returns an echo.HandlerFunc that handles WebSocket upgrade requests for a given connection type.
// Shutdown gracefully stops the bridge's background processes.
// The provided context is used for the shutdown timeout.
func (b *Bridge) Shutdown(ctx context.Context) {
	slog.Info("Shutting down WebSocket bridge", "endpoint", b.endpoint)
	if b.cancel != nil {
		b.cancel() // This will cause the pub/sub subscriptions to terminate
	}

	// Forcefully close all active client connections.
	b.clients.CloseAll()

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
			ID:       watermill.NewShortUUID(),
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
		client.Close() // Safely close the client's channel.
		
		// Publish client disconnected event
		go func() {
			payload, _ := json.Marshal(map[string]any{
				"userID":   client.UserID,
				"endpoint": client.Endpoint,
				"reason":   "connection_closed",
			})
			disconnectMsg := pubsub.Message{
				Topic:   TopicClientDisconnected.Name(),
				UserID:  client.UserID,
				Payload: payload,
			}
			if err := b.publisher.Publish(context.Background(), disconnectMsg); err != nil {
				slog.Error("Failed to publish websocket disconnect event", "error", err, "userID", client.UserID)
			}
		}()
		
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

		b.handleIncoming(client, message)
	}
}

func (b *Bridge) handleIncoming(client *Client, rawMsg []byte) {
	// Try to parse as a subscription message first
	var subMsg SubscribeMessage
	if err := json.Unmarshal(rawMsg, &subMsg); err == nil && (subMsg.Action == "subscribe" || subMsg.Action == "unsubscribe") {
		b.handleSubscription(client, subMsg)
		return
	}

	// Handle regular messages
	var msg struct {
		Action  string          `json:"action"`
		Topic   string          `json:"topic"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		slog.Warn("Received invalid message from client", "clientID", client.ID, "error", err)
		return // Ignore malformed messages
	}

	if msg.Action == "" {
		slog.Warn("Incoming message missing 'action' field", "clientID", client.ID)
		return
	}

	// Check if the action is whitelisted
	if !b.whitelist.IsAllowed(msg.Action) {
		slog.Warn("Client attempted to use non-whitelisted action",
			"clientID", client.ID,
			"action", msg.Action)
		return
	}

	// If a topic is not specified in the message, use the action as the topic.
	// This provides backward compatibility and a sensible default.
	if msg.Topic == "" {
		msg.Topic = msg.Action
	}

	// Verify the client is subscribed to the topic
	if !b.isClientSubscribed(client.ID, msg.Topic) {
		slog.Warn("Client attempted to publish to unsubscribed topic",
			"clientID", client.ID,
			"topic", msg.Topic)
		return
	}

	b.publisher.Publish(context.Background(), pubsub.Message{
		Topic:   msg.Topic,
		Payload: msg.Payload,
		UserID:  client.UserID,
	})
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

// handleSubscription manages topic subscriptions for a client
func (b *Bridge) handleSubscription(client *Client, msg SubscribeMessage) {
	topic := msg.Topic
	if msg.Payload.Channel != "" {
		topic = fmt.Sprintf("%s.%s", topic, msg.Payload.Channel)
	}

	switch msg.Action {
	case "subscribe":
		b.subscribeClient(client.ID, topic)
		slog.Info("Client subscribed to topic",
			"clientID", client.ID,
			"topic", topic)

	case "unsubscribe":
		b.unsubscribeClient(client.ID, topic)
		slog.Info("Client unsubscribed from topic",
			"clientID", client.ID,
			"topic", topic)
	}
}

// subscribeClient adds a client to a topic
func (b *Bridge) subscribeClient(clientID, topic string) {
	b.topics.Lock()
	defer b.topics.Unlock()

	if _, exists := b.topics.subscriptions[topic]; !exists {
		b.topics.subscriptions[topic] = make(map[string]struct{})
	}
	b.topics.subscriptions[topic][clientID] = struct{}{}
}

// unsubscribeClient removes a client from a topic
func (b *Bridge) unsubscribeClient(clientID, topic string) {
	b.topics.Lock()
	defer b.topics.Unlock()

	if subscribers, exists := b.topics.subscriptions[topic]; exists {
		delete(subscribers, clientID)
		if len(subscribers) == 0 {
			delete(b.topics.subscriptions, topic)
		}
	}
}

// isClientSubscribed checks if a client is subscribed to a topic
func (b *Bridge) isClientSubscribed(clientID, topic string) bool {
	b.topics.RLock()
	defer b.topics.RUnlock()

	subscribers, exists := b.topics.subscriptions[topic]
	if !exists {
		return false
	}
	_, subscribed := subscribers[clientID]
	return subscribed
}
