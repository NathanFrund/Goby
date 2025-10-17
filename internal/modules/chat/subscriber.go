package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nfrund/goby/internal/modules/chat/templates/components"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/websocket"
)

// ChatSubscriber listens for new chat messages on the pub/sub bus,
// renders them to HTML, and broadcasts them to all connected clients
// via the WebsocketBridge.
type ChatSubscriber struct {
	subscriber pubsub.Subscriber
	bridge     websocket.Bridge
	renderer   rendering.Renderer
}

// NewChatSubscriber creates a new subscriber service for the chat module.
func NewChatSubscriber(sub pubsub.Subscriber, bridge websocket.Bridge, renderer rendering.Renderer) *ChatSubscriber {
	return &ChatSubscriber{
		subscriber: sub,
		bridge:     bridge,
		renderer:   renderer,
	}
}

// Start begins listening for chat-related messages.
// This method blocks until the provided context is canceled.
// TODO: Consider adding metrics for message processing
func (cs *ChatSubscriber) Start(ctx context.Context) {
	slog.Info("Starting chat module subscriber")

	// Listen for new chat messages to broadcast
	go func() {
		err := cs.subscriber.Subscribe(ctx, "chat.messages.new", cs.handleNewMessage)
		if err != nil && err != context.Canceled {
			slog.Error("Chat message subscriber stopped with error", "error", err)
		}
	}()

	// Listen for direct chat messages
	go func() {
		err := cs.subscriber.Subscribe(ctx, "chat.messages.direct", cs.handleDirectMessage)
		if err != nil && err != context.Canceled {
			slog.Error("Direct message subscriber stopped with error", "error", err)
		}
	}()

	// Listen for new client connections to send a welcome message
	go func() {
		err := cs.subscriber.Subscribe(ctx, "system.websocket.connected", cs.handleClientConnect)
		if err != nil && err != context.Canceled {
			slog.Error("Chat client connect subscriber stopped with error", "error", err)
		}
	}()
}

// handleClientConnect sends a welcome message to a newly connected client.
func (cs *ChatSubscriber) handleClientConnect(ctx context.Context, msg pubsub.Message) error {
	var connectEvent struct {
		ConnectionType websocket.ConnectionType `json:"connectionType"`
	}
	if err := json.Unmarshal(msg.Payload, &connectEvent); err != nil {
		return err // Ignore malformed events
	}

	// Only send a welcome message to HTML clients.
	if connectEvent.ConnectionType == websocket.ConnectionTypeHTML {
		welcomeComponent := components.WelcomeMessage("Welcome to the chat, " + msg.UserID + "!")
		renderedHTML, err := cs.renderer.RenderComponent(ctx, welcomeComponent)
		if err == nil {
			// Send an HTML message directly to the user who just connected.
			// Using raw bytes from renderer for better performance
			message := &websocket.Message{
				Type:    "html",
				Target:  "#chat-messages",
				Payload: renderedHTML, // Will be properly encoded by Message.MarshalJSON
			}
			if err := cs.bridge.SendDirect(msg.UserID, message); err != nil {
				slog.Error("Failed to send welcome message", "error", err, "userID", msg.UserID)
			}
		}
	}
	return nil
}

// handleNewMessage processes incoming chat messages
func (cs *ChatSubscriber) handleNewMessage(ctx context.Context, msg pubsub.Message) error {
	var incoming struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal(msg.Payload, &incoming); err != nil {
		slog.Error("Failed to unmarshal chat message", "error", err, "payload", string(msg.Payload))
		return err
	}

	// Render the message to an HTML component
	component := components.ChatMessage(msg.UserID, incoming.Content, time.Now())
	renderedHTML, err := cs.renderer.RenderComponent(ctx, component)
	if err != nil {
		return fmt.Errorf("failed to render chat message: %w", err)
	}

	// Create and send the WebSocket message using raw bytes
	message := &websocket.Message{
		Type:    "html",
		Target:  "#chat-messages",
		Payload: renderedHTML, // Will be properly encoded by Message.MarshalJSON
	}
	return cs.bridge.Broadcast(message)
}

// handleDirectMessage processes direct chat messages
func (cs *ChatSubscriber) handleDirectMessage(ctx context.Context, msg pubsub.Message) error {
	var incoming struct {
		Content string `json:"content"`
		To      string `json:"to"`
	}

	if err := json.Unmarshal(msg.Payload, &incoming); err != nil {
		return fmt.Errorf("failed to unmarshal direct message: %w", err)
	}

	// Use ChatMessage with a prefix for direct messages
	component := components.ChatMessage(msg.UserID, "(DM) "+incoming.Content, time.Now())
	renderedHTML, err := cs.renderer.RenderComponent(ctx, component)
	if err != nil {
		return fmt.Errorf("failed to render direct message: %w", err)
	}

	// Create and send the WebSocket message using raw bytes
	message := &websocket.Message{
		Type:    "html",
		Target:  "#direct-messages",
		Payload: renderedHTML, // Will be properly encoded by Message.MarshalJSON
	}
	return cs.bridge.SendDirect(incoming.To, message)
}
