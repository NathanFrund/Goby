package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	announcerEvents "github.com/nfrund/goby/internal/modules/announcer/events"
	"github.com/nfrund/goby/internal/modules/chat/events"
	"github.com/nfrund/goby/internal/modules/chat/templates/components"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	wsTopics "github.com/nfrund/goby/internal/websocket"
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

	// Listen for new messages from clients.
	// These messages originate from clients via the websocket bridge.
	go func() {
		err := cs.subscriber.Subscribe(ctx, ClientMessageNew.Name(), cs.handleChatMessage)
		if err != nil && err != context.Canceled {
			slog.Error("Chat message subscriber stopped with error", "error", err)
		}
	}()

	// Also listen on the module's own "chat.messages" topic for messages
	// that might originate from other parts of the system (e.g., an HTTP handler).
	go func() {
		err := cs.subscriber.Subscribe(ctx, Messages.Name(), cs.handleChatMessage)
		if err != nil && err != context.Canceled {
			slog.Error("Chat message subscriber stopped with error", "error", err)
		}
	}()
	// Listen for new WebSocket connections to send welcome messages and subscribe to direct messages
	go func() {
		err := cs.subscriber.Subscribe(ctx, wsTopics.TopicClientReady.Name(), cs.handleClientConnect)
		if err != nil && err != context.Canceled {
			slog.Error("Chat client connect subscriber stopped with error", "error", err)
		}
	}()

	// Listen for user creation events from the announcer module
	go func() {
		err := cs.subscriber.Subscribe(ctx, "announcer.user.created", cs.handleUserCreated)
		if err != nil && err != context.Canceled {
			slog.Error("User created subscriber stopped with error", "error", err)
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
		directMsg := pubsub.Message{
			Topic:   wsTopics.TopicHTMLDirect.Name(),
			Payload: renderedHTML,
			Metadata: map[string]string{
				"recipient_id": readyEvent.UserID,
			},
		}
		return cs.publisher.Publish(ctx, directMsg)
	}

	return nil
}

// handleChatMessage processes incoming chat messages (both direct and broadcast)
func (cs *ChatSubscriber) handleChatMessage(ctx context.Context, msg pubsub.Message) error {
	// Parse the message payload using typed event
	var payload events.NewMessage

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
	isDirect := payload.Recipient != ""
	var recipient string
	if isDirect {
		// The recipient is now explicitly in the payload.
		recipient = payload.Recipient
	}

	// Determine the target topic based on message type
	var targetTopicName string
	if isDirect {
		targetTopicName = wsTopics.TopicHTMLDirect.Name()
	} else {
		targetTopicName = wsTopics.TopicHTMLBroadcast.Name()
	}

	// Publish the message with appropriate routing
	pubMsg := pubsub.Message{
		Topic:   targetTopicName,
		Payload: renderedHTML, // Send the raw rendered HTML directly
	}

	// Add recipient metadata for direct messages
	if isDirect {
		pubMsg.Metadata = map[string]string{
			"recipient_id": recipient,
		}
	}

	return cs.publisher.Publish(ctx, pubMsg)
}

// handleUserCreated processes user creation events from the announcer module
func (cs *ChatSubscriber) handleUserCreated(ctx context.Context, msg pubsub.Message) error {
	var eventData announcerEvents.UserCreated

	if err := json.Unmarshal(msg.Payload, &eventData); err != nil {
		slog.Error("Failed to unmarshal user created event", "error", err)
		return nil // Don't stop the subscriber for a bad message
	}

	// Create a broadcast message announcing the new user
	content := "🎉 " + eventData.Email + " has joined!"
	announcement := struct {
		Content string `json:"content"`
		User    string `json:"user"`
	}{
		Content: content,
		User:    "system", // System-generated announcement
	}

	payload, err := json.Marshal(announcement)
	if err != nil {
		slog.Error("Failed to marshal user join announcement", "error", err)
		return err
	}

	// Use the existing handleChatMessage method to process and broadcast this message
	announcementMsg := pubsub.Message{
		Topic:   Messages.Name(), // Send to the chat messages topic
		Payload: payload,
		UserID:  "system",
	}

	return cs.handleChatMessage(ctx, announcementMsg)
}
