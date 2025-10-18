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
)

// ChatSubscriber listens for new chat messages on the pub/sub bus,
// renders them to HTML, and broadcasts them to all connected clients
// via the WebsocketBridge.
type ChatSubscriber struct {
	subscriber pubsub.Subscriber
	publisher  pubsub.Publisher
	renderer   rendering.Renderer
}

// NewChatSubscriber creates a new subscriber service for the chat module.
func NewChatSubscriber(sub pubsub.Subscriber, pub pubsub.Publisher, renderer rendering.Renderer) *ChatSubscriber {
	return &ChatSubscriber{
		subscriber: sub,
		publisher:  pub,
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
		err := cs.subscriber.Subscribe(ctx, "system.websocket.ready", cs.handleClientConnect)
		if err != nil && err != context.Canceled {
			slog.Error("Chat client connect subscriber stopped with error", "error", err)
		}
	}()
}

// handleClientConnect sends a welcome message to a newly connected client.
func (cs *ChatSubscriber) handleClientConnect(ctx context.Context, msg pubsub.Message) error {
	var readyEvent struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(msg.Payload, &readyEvent); err != nil {
		slog.Error("Failed to unmarshal system.websocket.ready event", "error", err)
		return nil // Don't stop the subscriber for a bad message
	}

	// Only send a welcome message to HTML clients.
	if readyEvent.Endpoint == "html" {
		welcomeComponent := components.WelcomeMessage("Welcome to the chat, " + msg.UserID + "!")
		renderedHTML, err := cs.renderer.RenderComponent(ctx, welcomeComponent)
		if err != nil {
			slog.Error("Failed to render welcome message", "error", err, "userID", msg.UserID)
			return err
		}

		// Publish the welcome message to the user's direct HTML topic.
		return cs.publisher.Publish(ctx, pubsub.Message{
			Topic:    "ws.html.direct",
			Payload:  renderedHTML,
			Metadata: map[string]string{"user_id": msg.UserID},
		})
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

	// Broadcast the rendered HTML to all connected HTML clients.
	return cs.publisher.Publish(ctx, pubsub.Message{
		Topic:   "ws.html.broadcast",
		Payload: renderedHTML,
	})
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
	return cs.publisher.Publish(ctx, pubsub.Message{
		Topic:   "ws.html.direct",
		Payload: renderedHTML,
		Metadata: map[string]string{
			"user_id": incoming.To,
		},
	})
}
