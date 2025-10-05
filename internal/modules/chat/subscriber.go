package chat

import (
	"context"
	"encoding/json"
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
	bridge     *websocket.WebsocketBridge
	renderer   rendering.Renderer
}

// NewChatSubscriber creates a new subscriber service for the chat module.
func NewChatSubscriber(sub pubsub.Subscriber, bridge *websocket.WebsocketBridge, renderer rendering.Renderer) *ChatSubscriber {
	return &ChatSubscriber{
		subscriber: sub,
		bridge:     bridge,
		renderer:   renderer,
	}
}

// Start begins listening for messages on the "chat.messages.new" topic.
// This method blocks until the provided context is canceled.
func (cs *ChatSubscriber) Start(ctx context.Context) {
	slog.Info("Starting chat module subscriber")
	err := cs.subscriber.Subscribe(ctx, "chat.messages.new", cs.handleNewMessage)
	if err != nil && err != context.Canceled {
		slog.Error("Chat module subscriber stopped with error", "error", err)
	}
	slog.Info("Chat module subscriber stopped")
}

// handleNewMessage is the handler function for incoming pub/sub messages.
func (cs *ChatSubscriber) handleNewMessage(ctx context.Context, msg pubsub.Message) error {
	var incoming struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(msg.Payload, &incoming); err != nil {
		slog.Error("Failed to unmarshal chat message payload", "error", err, "payload", string(msg.Payload))
		return err // Returning an error will Nack the message.
	}

	// Render the message to an HTML component.
	// In a real app, you might fetch the username from the msg.UserID.
	component := components.ChatMessage(msg.UserID, incoming.Content, time.Now())
	renderedHTML, err := cs.renderer.RenderComponent(ctx, component)
	if err != nil {
		slog.Error("Failed to render chat message component", "error", err)
		return err
	}

	// Broadcast the final HTML to all clients via the bridge.
	cs.bridge.BroadcastToAll(renderedHTML)
	return nil
}
