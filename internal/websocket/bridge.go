package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/pubsub"
)

// ConnectionType defines the type of WebSocket connection.
type ConnectionType int

const (
	// ConnectionTypeHTML is for clients that consume HTML fragments (e.g., HTMX).
	ConnectionTypeHTML ConnectionType = iota
	// ConnectionTypeData is for clients that consume structured data (e.g., JSON).
	ConnectionTypeData
)

// String returns a string representation of the ConnectionType
func (t ConnectionType) String() string {
	switch t {
	case ConnectionTypeHTML:
		return "html"
	case ConnectionTypeData:
		return "data"
	default:
		return "unknown"
	}
}

// Client represents a single connected WebSocket client in the bridge.
type Client struct {
	// ID is the unique identifier for the client, typically the User ID.
	ID string
	// conn is the underlying WebSocket connection.
	conn *websocket.Conn
	// send is a buffered channel of outbound messages for this client.
	send chan []byte
	// connType is the type of connection (HTML or Data).
	connType ConnectionType
	// bridge is a reference back to the bridge that manages this client.
	bridge *bridge
	// mu protects the send channel during client shutdown
	mu sync.RWMutex
}

// IncomingMessage represents a message received from a client, destined for the pub/sub system.
type IncomingMessage struct {
	ClientID string
	Payload  []byte
	Topic    string
}

// Message is defined in message.go

// BroadcastMessage represents a message to be broadcast to clients.
type BroadcastMessage struct {
	payload []byte
	// targetTypes specifies which connection types should receive the message.
	targetTypes map[ConnectionType]bool
}

// Bridge defines the interface for the WebSocket manager.
type Bridge interface {
	Run()
	Handler(connType ConnectionType) echo.HandlerFunc
	Broadcast(msg *Message) error
	SendDirect(userID string, msg *Message) error
	SendCommand(userID string, cmdName string, payload ...interface{}) error
	SendHTML(userID string, html string, target string) error
	SendData(userID string, data interface{}) error
}

// bridge manages all WebSocket connections and routes messages
// between connected clients and the Pub/Sub message bus.
type bridge struct {
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber

	// clients is a map of user IDs to a list of their active clients.
	// A user can have multiple connections (e.g., browser tab, mobile).
	clients map[string][]*Client

	// register is a channel for new clients to register.
	register chan *Client

	// unregister is a channel for clients to unregister.
	unregister chan *Client

	// broadcast is a channel for messages to be sent to all relevant clients.
	broadcast chan *BroadcastMessage

	// incoming is a channel for messages received from clients.
	incoming chan *IncomingMessage

	// A mutex to protect access to the clients map and cancelFuncs
	mu sync.RWMutex

	// cancelFuncs stores cancel functions for active subscriptions
	cancelFuncs map[string]context.CancelFunc
}

// NewBridge initializes a new Bridge, ready to handle connections.
func NewBridge(pub pubsub.Publisher, sub pubsub.Subscriber) (Bridge, error) {
	b := &bridge{
		publisher:   pub,
		subscriber:  sub,
		clients:     make(map[string][]*Client),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *BroadcastMessage, 100),
		incoming:    make(chan *IncomingMessage, 100),
		cancelFuncs: make(map[string]context.CancelFunc),
	}

	// Start a permanent subscription for broadcast messages
	go func() {
		err := b.subscriber.Subscribe(context.Background(), "broadcast", b.handleBroadcast)
		if err != nil {
			slog.Error("Failed to subscribe to broadcast topic, broadcast will not work", "error", err)
		}
	}()

	return b, nil
}

// Run starts the main bridge goroutine for managing client lifecycle and message routing.
func (b *bridge) Run() {
	slog.Info("New WebSocket bridge runner started.")
	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			// Add the new client to the map.
			b.clients[client.ID] = append(b.clients[client.ID], client)
			b.mu.Unlock()
			// Use Debug level for successful registration to reduce log noise.
			slog.Debug("Client registered to bridge", "userID", client.ID, "type", client.connType)
		case client := <-b.unregister:
			b.mu.Lock()
			// Remove the client from the list of clients for that user.
			if clients, ok := b.clients[client.ID]; ok {
				for i, c := range clients {
					if c == client {
						b.clients[client.ID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}

				// If this was the last client for this user, clean up
				if len(b.clients[client.ID]) == 0 {
					delete(b.clients, client.ID)
					// Cancel the subscription if it exists
					if cancel, ok := b.cancelFuncs[client.ID]; ok {
						cancel()
						delete(b.cancelFuncs, client.ID)
						slog.Debug("Cancelled subscription for user", "userID", client.ID)
					}
				}

				// Safely close the client's send channel.
				// This must be done while holding the lock to prevent race conditions
				// where a message is sent just as the client is being unregistered.
				client.mu.Lock()
				if client.send != nil {
					close(client.send)
					client.send = nil // Prevent further writes
				}
				client.mu.Unlock()
			}
			b.mu.Unlock()
			slog.Debug("Client unregistered", "userID", client.ID, "type", client.connType)

		case msg := <-b.broadcast:
			b.mu.RLock()
			for _, clients := range b.clients {
				for _, client := range clients {
					// Check if the client's connection type is one of the targets.
					if !msg.targetTypes[client.connType] {
						continue
					}
					select {
					case client.send <- msg.payload:
					default:
						// Drop message if client's send buffer is full.
						slog.Warn("Client send channel full, dropping message", "userID", client.ID)
					}
				}
			}
			b.mu.RUnlock()

		case msg := <-b.incoming:
			// Dynamically route incoming messages based on a topic field in the payload.
			// We only unmarshal to get the topic, then pass the original payload on.
			var routedMessage struct {
				Topic string `json:"topic"`
			}
			if err := json.Unmarshal(msg.Payload, &routedMessage); err != nil {
				slog.Warn("Failed to unmarshal incoming message for routing", "error", err, "payload", string(msg.Payload))
				continue
			}

			if routedMessage.Topic == "" {
				slog.Warn("Incoming message missing 'topic' field", "payload", string(msg.Payload))
				continue
			}

			pubsubMsg := pubsub.Message{
				Topic:   routedMessage.Topic,
				UserID:  msg.ClientID,
				Payload: msg.Payload, // Pass the original, full payload
			}

			if err := b.publisher.Publish(context.Background(), pubsubMsg); err != nil {
				slog.Error("Bridge failed to publish incoming message", "userID", msg.ClientID, "topic", routedMessage.Topic, "error", err)
			}
		}
	}
}

// handleBroadcast is a callback for the pub/sub system that forwards broadcast messages
// to all connected clients.
func (b *bridge) handleBroadcast(_ context.Context, msg pubsub.Message) error {
	// Unmarshal the message to determine its type and filter clients accordingly.
	var wsMsg Message
	if err := json.Unmarshal(msg.Payload, &wsMsg); err != nil {
		slog.Error("Failed to unmarshal broadcast message from pub/sub", "error", err)
		// Don't return an error, just log it and move on.
		// Returning an error might cause the subscription to terminate.
		return nil
	}

	// Validate the message type before processing
	if !isValidMessageType(wsMsg.Type) {
		slog.Warn("Received broadcast message with invalid type", "type", wsMsg.Type)
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, clients := range b.clients {
		for _, client := range clients {
			// Check if the client's connection type matches the message type
			// Commands are sent to all clients.
			if wsMsg.Type == MessageType(client.connType.String()) || wsMsg.Type == MessageTypeCommand {
				// For HTML messages, we send the raw HTML string payload
				// For Data/Command messages, we send the JSON-serialized payload
				var payloadToSend []byte
				var err error

				switch wsMsg.Type {
				case MessageTypeHTML:
					if htmlPayload, ok := wsMsg.Payload.(string); ok {
						payloadToSend = []byte(htmlPayload)
					} else {
						slog.Warn("HTML message payload is not a string", "payload", wsMsg.Payload)
						continue
					}

				case MessageTypeData, MessageTypeCommand:
					// For data and command messages, marshal just the payload
					payloadToSend, err = json.Marshal(wsMsg.Payload)
					if err != nil {
						slog.Error("Failed to marshal message payload",
							"error", err,
							"type", wsMsg.Type,
							"payload", wsMsg.Payload)
						continue
					}

				default:
					slog.Warn("Unhandled message type", "type", wsMsg.Type)
					continue
				}

				client.sendMessage(payloadToSend)
			}
		}
	}
	return nil
}

// Handler returns an echo.HandlerFunc that handles WebSocket upgrade requests for a given connection type.
func (b *bridge) Handler(connType ConnectionType) echo.HandlerFunc {
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
			ID:       user.Email, // Using email as the unique ID for now.
			conn:     conn,
			send:     make(chan []byte, 256),
			connType: connType,
			bridge:   b,
		}

		// Register the client
		b.register <- client

		// Create a context for this client's subscription
		ctx, cancel := context.WithCancel(context.Background())

		b.mu.Lock()
		// Store the cancel function for cleanup
		b.cancelFuncs[user.Email] = cancel
		b.mu.Unlock()

		// Start a goroutine to handle the subscription
		go func() {
			// Subscribe to the user's direct message topic
			topic := "direct." + user.Email
			err := b.subscriber.Subscribe(ctx, topic, func(ctx context.Context, msg pubsub.Message) error {
				var wsMsg Message
				if err := json.Unmarshal(msg.Payload, &wsMsg); err != nil {
					slog.Error("Failed to unmarshal WebSocket message", "error", err, "userID", user.Email)
					return nil
				}

				// Validate the message type
				if !isValidMessageType(wsMsg.Type) {
					slog.Warn("Received direct message with invalid type", "type", wsMsg.Type, "userID", user.Email)
					return nil
				}

				// Check if the client's connection type matches the message type
				// Commands are sent to all clients.
				if wsMsg.Type == MessageType(client.connType.String()) || wsMsg.Type == MessageTypeCommand {
					var payloadToSend []byte
					var err error

					switch wsMsg.Type {
					case MessageTypeHTML:
						if htmlPayload, ok := wsMsg.Payload.(string); ok {
							payloadToSend = []byte(htmlPayload)
						} else {
							slog.Warn("HTML message payload is not a string",
								"userID", user.Email,
								"payload", wsMsg.Payload)
							return nil
						}

					case MessageTypeData, MessageTypeCommand:
						// For data and command messages, marshal just the payload
						payloadToSend, err = json.Marshal(wsMsg.Payload)
						if err != nil {
							slog.Error("Failed to marshal direct message payload",
								"error", err,
								"userID", user.Email,
								"type", wsMsg.Type)
							return nil
						}

					default:
						slog.Warn("Unhandled direct message type",
							"userID", user.Email,
							"type", wsMsg.Type)
						return nil
					}

					client.sendMessage(payloadToSend)
				}

				return nil
			})

			if err != nil && err != context.Canceled {
				slog.Error("Subscription error", "userID", user.Email, "error", err)
			}
		}()

		// Start the read and write pumps
		go client.writePump()
		go client.readPump()

		// Publish a generic "client connected" event to the message bus.
		go func() {
			payload, _ := json.Marshal(map[string]any{
				"userID":         user.Email,
				"connectionType": connType,
			})
			connectMsg := pubsub.Message{
				Topic:   "system.websocket.connected",
				UserID:  user.Email,
				Payload: payload,
			}
			if err := b.publisher.Publish(context.Background(), connectMsg); err != nil {
				slog.Error("Failed to publish websocket connect event", "error", err)
			}
		}()

		return nil
	}
}

// readPump pumps messages from the WebSocket connection to the bridge's incoming channel.
func (c *Client) readPump() {
	// Ensure the client is unregistered when the readPump exits for any reason
	// (e.g., connection closed by client, read error).
	defer func() {
		c.bridge.unregister <- c
		c.conn.Close(websocket.StatusNormalClosure, "Client disconnected")
	}()

	for {
		_, message, err := c.conn.Read(context.Background())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				slog.Info("WebSocket closed normally by client", "userID", c.ID)
			} else if err != io.EOF {
				slog.Error("WebSocket read error", "userID", c.ID, "error", err)
			}
			break
		}

		// Forward the message to the bridge's central incoming channel.
		c.bridge.incoming <- &IncomingMessage{
			ClientID: c.ID,
			Payload:  message,
		}
	}
}

// sendMessage safely sends a message to the client's send channel, logging if it's full.
func (c *Client) sendMessage(message []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.send == nil {
		// This can happen if the client is disconnected and the channel is closed.
		// Log at debug level as this is an expected condition during shutdown.
		slog.Debug("Client send channel is nil, message dropped",
			"userID", c.ID,
			"connType", c.connType)
		return
	}

	select {
	case c.send <- message:
	// Message sent successfully
	default:
		slog.Warn("Client send channel full, dropping message", "userID", c.ID, "connType", c.connType)
	}
}

// writePump pumps messages from the client's send channel to the WebSocket connection.
func (c *Client) writePump() {
	defer func() {
		// Close the WebSocket connection
		c.conn.Close(websocket.StatusNormalClosure, "Server-side cleanup")
	}()

	for message := range c.send {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := c.conn.Write(ctx, websocket.MessageText, message)
		cancel()
		if err != nil {
			slog.Error("WebSocket write error", "userID", c.ID, "error", err)
			return
		}
	}
}

// Incoming returns the channel for messages received from clients.
func (b *bridge) Incoming() <-chan *IncomingMessage {
	return b.incoming
}

// SendDirect sends a message directly to a specific user.
// The message will be delivered to the user's active connections.
func (b *bridge) SendDirect(userID string, msg *Message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Failed to marshal direct message", "error", err, "userID", userID)
		return fmt.Errorf("failed to marshal direct message: %w", err)
	}

	topic := "direct." + userID
	return b.publisher.Publish(context.Background(), pubsub.Message{
		Topic:   topic,
		Payload: msgBytes,
	})
}

// SendCommand sends a command message to a specific user
func (b *bridge) SendCommand(userID string, cmdName string, payload ...interface{}) error {
	return b.SendDirect(userID, NewCommand(cmdName, payload...))
}

// SendHTML sends an HTML message to a specific user
func (b *bridge) SendHTML(userID string, html string, target string) error {
	return b.SendDirect(userID, NewHTMLMessage(html, target))
}

// SendData sends a data message to a specific user
func (b *bridge) SendData(userID string, data interface{}) error {
	return b.SendDirect(userID, NewDataMessage(data))
}

// Broadcast sends a message to all connected users
func (b *bridge) Broadcast(msg *Message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Failed to marshal broadcast message", "error", err, "messageType", msg.Type)
		return fmt.Errorf("failed to marshal broadcast message: %w", err)
	}

	return b.publisher.Publish(context.Background(), pubsub.Message{
		Topic:   "broadcast",
		Payload: msgBytes,
	})
}
