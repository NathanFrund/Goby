package websocket

import (
	"context"
	"encoding/json"
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
}

// IncomingMessage represents a message received from a client, destined for the pub/sub system.
type IncomingMessage struct {
	ClientID string
	Payload  []byte
	Topic    string
}

// BroadcastMessage represents a message to be broadcast to clients.
type BroadcastMessage struct {
	payload []byte
	// targetTypes specifies which connection types should receive the message.
	targetTypes map[ConnectionType]bool
}

// DirectMessage represents a message to be sent to a single user.
type DirectMessage struct {
	TargetUserID string
	Payload      []byte
	// targetTypes specifies which connection types should receive the message.
	targetTypes map[ConnectionType]bool
}

// Bridge defines the interface for the WebSocket manager.
type Bridge interface {
	Run()
	Handler(connType ConnectionType) echo.HandlerFunc
	Broadcast(payload []byte, connTypes ...ConnectionType)
	SendDirect(userID string, payload []byte, connTypes ...ConnectionType)
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

	// direct is a channel for messages to be sent to a specific user.
	direct chan *DirectMessage

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
		direct:      make(chan *DirectMessage, 100),
		incoming:    make(chan *IncomingMessage, 100),
		cancelFuncs: make(map[string]context.CancelFunc),
	}

	return b, nil
}


// Run starts the main bridge goroutine for managing client lifecycle and message routing.
func (b *bridge) Run() {
	slog.Info("New WebSocket bridge runner started.")
	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client.ID] = append(b.clients[client.ID], client)
			b.mu.Unlock()
			slog.Info("Client registered to new bridge", "userID", client.ID, "type", client.connType)

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
			}
			close(client.send)
			b.mu.Unlock()
			slog.Info("Client unregistered", "userID", client.ID, "type", client.connType)

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

		case message := <-b.direct:
			b.mu.RLock()
			if clients, ok := b.clients[message.TargetUserID]; ok {
				for _, client := range clients {
					// Check if the client's connection type is one of the targets.
					if !message.targetTypes[client.connType] {
						continue
					}
					select {
					case client.send <- message.Payload:
					default:
						slog.Warn("Client send channel full, dropping direct message", "userID", client.ID)
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
			slog.Error("Failed to upgrade connection to WebSocket", "error", err)
			return err
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
				// Forward the message to the client
				select {
				case client.send <- msg.Payload:
				default:
					slog.Warn("Client send channel full, dropping message", "userID", user.Email)
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

// writePump pumps messages from the client's send channel to the WebSocket connection.
func (c *Client) writePump() {
	defer func() {
		c.conn.Close(websocket.StatusNormalClosure, "Server-side cleanup")
	}()

	for {
		message, ok := <-c.send
		if !ok {
			// The bridge closed the channel.
			return
		}

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

// Broadcast sends a message to all clients of the specified connection types.
func (b *bridge) Broadcast(payload []byte, connTypes ...ConnectionType) {
	targets := make(map[ConnectionType]bool)
	for _, t := range connTypes {
		targets[t] = true
	}

	b.broadcast <- &BroadcastMessage{
		payload:     payload,
		targetTypes: targets,
	}
}

// SendDirect sends a message directly to a specific user by publishing it to the message bus.
// The message will be delivered to all of the user's active connections.
// The payload can be either a raw string (for HTML) or a JSON-encoded value (for data connections).
func (b *bridge) SendDirect(userID string, payload []byte, connTypes ...ConnectionType) {
	// Publish to the user's direct message topic
	topic := "direct." + userID
	err := b.publisher.Publish(context.Background(), pubsub.Message{
		Topic:   topic,
		Payload: payload,
	})

	if err != nil {
		slog.Error("Failed to publish direct message", "error", err, "userID", userID, "topic", topic)
	}
}