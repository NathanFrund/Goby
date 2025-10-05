package websocket

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/nfrund/goby/internal/pubsub"

	"github.com/coder/websocket"
)

// Client represents a single connected client. We will use a map
// of these to manage active connections.
type Client struct {
	// ID is the unique identifier for the client, often the User ID.
	ID string
	// conn is the actual WebSocket connection.
	conn *websocket.Conn
	// send is a channel of outbound messages to the client.
	send chan []byte
}

// WebsocketBridge manages all WebSocket connections and routes messages
// between connected clients and the Pub/Sub message bus.
type WebsocketBridge struct {
	publisher  pubsub.Publisher
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
}

// NewWebsocketBridge initializes a new WebsocketBridge, ready to handle connections.
func NewWebsocketBridge(pub pubsub.Publisher) *WebsocketBridge {
	return &WebsocketBridge{
		publisher:  pub,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// ServeHTTP handles incoming WebSocket upgrade requests and sets up the client.
// It implements the http.Handler interface.
func (wb *WebsocketBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// In a real application, you would extract the authenticated User ID here
	// from the request context or session data.
	// For now, we'll use a placeholder user ID derived from the connection time.
	// NOTE: This MUST be replaced with real authentication in production.
	userID := "user_" + time.Now().Format("150405")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// In a production environment, you should check the origin to prevent CSRF.
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("Failed to upgrade connection to WebSocket", "error", err)
		return
	}

	client := &Client{ID: userID, conn: conn, send: make(chan []byte, 256)}
	wb.register <- client

	// Start goroutines for reading (incoming) and writing (outgoing) messages.
	// The writePump is now responsible for pinging, so we only need two goroutines.
	go wb.writePump(client)
	go wb.readPump(client)
}

// Run starts the main bridge goroutine for managing client lifecycle events.
func (wb *WebsocketBridge) Run() {
	slog.Info("Websocket bridge runner started.")
	for {
		select {
		case client := <-wb.register:
			wb.clients[client] = true
			slog.Info("Client registered", "userID", client.ID)

		case client := <-wb.unregister:
			if _, ok := wb.clients[client]; ok {
				delete(wb.clients, client)
				close(client.send)
				slog.Info("Client unregistered", "userID", client.ID)
			}
		}
	}
}

// HandleIncomingMessage publishes any message received from a client
// to the Pub/Sub system.
func (wb *WebsocketBridge) HandleIncomingMessage(client *Client, message []byte) {
	// In a real app, you would parse the message to determine the target topic.
	// For now, we assume all messages go to a generic "chat" topic.
	msg := pubsub.Message{
		Topic:   "chat.messages.new",
		UserID:  client.ID,
		Payload: message,
		Metadata: map[string]string{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}

	err := wb.publisher.Publish(context.Background(), msg)
	if err != nil {
		slog.Error("Failed to publish incoming client message", "userID", client.ID, "error", err)
	}
}

// readPump reads messages from the WebSocket connection and sends them to the publisher.
func (wb *WebsocketBridge) readPump(client *Client) {
	defer func() {
		wb.unregister <- client
		client.conn.Close(websocket.StatusNormalClosure, "Client disconnected")
	}()

	// The coder/websocket library handles keep-alives automatically.
	// A read will fail if the connection is dead. We just need a simple loop.

	for {
		// We use a background context because read deadlines are managed by the
		// underlying library's keep-alive mechanism.
		_, message, err := client.conn.Read(context.Background())
		if err != nil {
			// Check if the error is a normal closure.
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
				slog.Info("WebSocket closed normally by client", "userID", client.ID)
			} else if err != io.EOF {
				slog.Error("WebSocket read error", "userID", client.ID, "error", err)
			}
			break
		}
		// Send the raw message content to the Pub/Sub system
		wb.HandleIncomingMessage(client, message)
	}
}

// writePump sends messages from the bridge back to the WebSocket connection.
func (wb *WebsocketBridge) writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close(websocket.StatusNormalClosure, "Server-side cleanup")
	}()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				// The bridge closed the channel (unregister signal)
				// The defer will handle the final conn.Close()
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			err := client.conn.Write(ctx, websocket.MessageText, message)
			cancel() // Release context resources
			if err != nil {
				slog.Error("WebSocket write error", "userID", client.ID, "error", err)
				return
			}

		case <-ticker.C:
			// The coder/websocket library handles pings automatically.
			// This ticker is no longer needed for keep-alives.
			// We can remove it to simplify the write pump.
			// If you needed to send application-level pings, you would do it here.
		}
	}
}

// WriteToClient is the external method used to push a message (received from Pub/Sub)
// out to a specific client identified by ID.
func (wb *WebsocketBridge) WriteToClient(userID string, message []byte) {
	// This method is useful for private messages or acknowledgements.
	// For a real-world scenario with many clients, this map should be keyed by userID
	// for O(1) lookups instead of O(N) iteration.
	for client := range wb.clients {
		if client.ID == userID {
			select {
			case client.send <- message:
				// Successfully sent to the client's queue
			default:
				// Client queue is full, assume connection is jammed/broken
				close(client.send)
				delete(wb.clients, client)
				slog.Warn("Client send channel full, connection dropped", "userID", userID)
			}
			return // Assuming one connection per user for simplicity
		}
	}
	slog.Debug("Attempted to write to non-existent client", "userID", userID)
}

// BroadcastToAll sends a message to all currently connected clients.
// This is essential for public chat room functionality.
func (wb *WebsocketBridge) BroadcastToAll(message []byte) {
	slog.Debug("Broadcasting message to all clients", "size", len(message))
	for client := range wb.clients {
		select {
		case client.send <- message:
			// Successfully sent to the client's queue
		default:
			// Client queue is full, assume connection is jammed/broken
			close(client.send)
			delete(wb.clients, client)
			slog.Warn("Client send channel full, connection dropped during broadcast", "userID", client.ID)
		}
	}
}

// --- Configuration Constants ---
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)
