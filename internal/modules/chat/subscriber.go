package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/nfrund/goby/internal/modules/chat/templates/components"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	wsTopics "github.com/nfrund/goby/internal/topics/websocket"
)

// ChatSubscriber listens for new chat messages on the pub/sub bus,
// renders them to HTML, and broadcasts them to all connected clients
// via the WebSocket bridge.
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
func (cs *ChatSubscriber) Start(ctx context.Context) {
	slog.Info("Starting chat module subscriber")

	// Listen for broadcast chat messages
	go func() {
		err := cs.subscriber.Subscribe(ctx, "chat.messages", cs.handleChatMessage)
		if err != nil && err != context.Canceled {
			slog.Error("Chat message subscriber stopped with error", "error", err)
		}
	}()

	// Listen for direct chat messages (wildcard subscription)
	go func() {
		err := cs.subscriber.Subscribe(ctx, "chat.direct.*", cs.handleChatMessage)
		if err != nil && err != context.Canceled {
			slog.Error("Direct message subscriber stopped with error", "error", err)
		}
	}()

	// Listen for new WebSocket connections to send welcome messages and subscribe to direct messages
	go func() {
		err := cs.subscriber.Subscribe(ctx, wsTopics.ClientReady.Name(), cs.handleClientConnect)
		if err != nil && err != context.Canceled {
			slog.Error("Chat client connect subscriber stopped with error", "error", err)
		}
	}()
}

// handleClientConnect sends a welcome message to a newly connected client.
func (cs *ChatSubscriber) handleClientConnect(ctx context.Context, msg pubsub.Message) error {
	var readyEvent struct {
		Endpoint string `json:"endpoint"`
		UserID   string `json:"userID"`
	}
	if err := json.Unmarshal(msg.Payload, &readyEvent); err != nil {
		slog.Error("Failed to unmarshal system.websocket.ready event", "error", err)
		return nil // Don't stop the subscriber for a bad message
	}

	// Only send a welcome message to HTML clients.
	if readyEvent.Endpoint == "html" && readyEvent.UserID != "" {
		welcomeComponent := components.WelcomeMessage("Welcome to the chat, " + readyEvent.UserID + "!")
		renderedHTML, err := cs.renderer.RenderComponent(ctx, welcomeComponent)
		if err != nil {
			slog.Error("Failed to render welcome message", "error", err, "userID", readyEvent.UserID)
			return err
		}

		// Publish the welcome message to the direct messages topic
		// Using metadata for recipient ID
		return cs.publisher.Publish(ctx, pubsub.Message{
			Topic:   "ws.html.direct",
			Payload: renderedHTML,
			Metadata: map[string]string{
				"recipient_id": readyEvent.UserID,
			},
		})
	}

	return nil
}

// handleChatMessage processes incoming chat messages (both direct and broadcast)
func (cs *ChatSubscriber) handleChatMessage(ctx context.Context, msg pubsub.Message) error {
	// Parse the message payload
	var payload struct {
		Content   string `json:"content"`
		User      string `json:"user"`
		Recipient string `json:"recipient,omitempty"`
	}

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		slog.Error("Failed to unmarshal chat message", "error", err)
		return nil // Don't stop the subscriber for a bad message
	}

	// Use the user from the payload if available, fallback to the message user ID
	userID := payload.User
	if userID == "" {
		userID = msg.UserID
	}

	// Render the message component with current timestamp
	messageComponent := components.ChatMessage(userID, payload.Content, time.Now())
	renderedHTML, err := cs.renderer.RenderComponent(ctx, messageComponent)
	if err != nil {
		slog.Error("Failed to render chat message", "error", err, "userID", userID)
		return err
	}

	// Determine if this is a direct message by checking the topic
	isDirect := strings.HasPrefix(msg.Topic, "chat.direct.")
	var recipient string
	if isDirect {
		// Extract recipient from the topic (format: chat.direct.user@example.com)
		recipient = strings.TrimPrefix(msg.Topic, "chat.direct.")
	}

	// Create a message for the WebSocket bridge
	wsMessage := struct {
		Topic   string `json:"topic"`
		Payload string `json:"payload"`
	}{
		Topic:   "chat.message",
		Payload: string(renderedHTML),
	}

	wsPayload, err := json.Marshal(wsMessage)
	if err != nil {
		slog.Error("Failed to marshal WebSocket message", "error", err)
		return err
	}

	// Determine the target topic based on message type
	var targetTopic string
	if isDirect {
		targetTopic = "ws.html.direct"
	} else {
		targetTopic = "ws.html.broadcast"
	}

	// Publish the message with appropriate routing
	pubMsg := pubsub.Message{
		Topic:   targetTopic,
		Payload: wsPayload,
	}

	// Add recipient metadata for direct messages
	if isDirect {
		pubMsg.Metadata = map[string]string{
			"recipient_id": recipient,
		}
	}

	return cs.publisher.Publish(ctx, pubMsg)
}
