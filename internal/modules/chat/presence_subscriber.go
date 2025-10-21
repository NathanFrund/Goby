package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nfrund/goby/internal/modules/chat/templates/components"
	"github.com/nfrund/goby/internal/presence"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	wsTopics "github.com/nfrund/goby/internal/websocket"
)

// PresenceSubscriber listens for presence updates and renders HTML fragments
type PresenceSubscriber struct {
	subscriber pubsub.Subscriber
	publisher  pubsub.Publisher
	renderer   rendering.Renderer
	logger     *slog.Logger
}

// NewPresenceSubscriber creates a new presence subscriber
func NewPresenceSubscriber(subscriber pubsub.Subscriber, publisher pubsub.Publisher, renderer rendering.Renderer) *PresenceSubscriber {
	return &PresenceSubscriber{
		subscriber: subscriber,
		publisher:  publisher,
		renderer:   renderer,
		logger:     slog.Default().With("component", "chat_presence_subscriber"),
	}
}

// Start begins listening for presence updates
func (ps *PresenceSubscriber) Start(ctx context.Context) {
	ps.logger.Info("Starting presence subscriber")

	// Subscribe to presence updates using the handler pattern
	err := ps.subscriber.Subscribe(ctx, presence.TopicUserStatusUpdate.Name(), ps.handlePresenceUpdate)
	if err != nil {
		ps.logger.Error("Failed to subscribe to presence updates", "error", err)
		return
	}

	ps.logger.Info("Successfully subscribed to presence updates")

	// Keep the subscriber running
	<-ctx.Done()
	ps.logger.Info("Presence subscriber shutting down")
}

// handlePresenceUpdate processes a presence update and publishes HTML
func (ps *PresenceSubscriber) handlePresenceUpdate(ctx context.Context, msg pubsub.Message) error {
	ps.logger.Info("Received presence update")

	// Parse the presence update
	var update struct {
		Type  string   `json:"type"`
		Users []string `json:"users"`
	}

	if err := json.Unmarshal(msg.Payload, &update); err != nil {
		ps.logger.Error("Failed to unmarshal presence update", "error", err)
		// Don't return error for malformed messages - just skip them
		return nil
	}

	ps.logger.Info("Processing presence update", "user_count", len(update.Users))

	// Render the presence component with retry logic
	var renderedHTML []byte
	var err error

	for attempt := 1; attempt <= 3; attempt++ {
		component := components.OnlineUsers(update.Users)
		renderedHTML, err = ps.renderer.RenderComponent(ctx, component)
		if err == nil {
			break
		}

		ps.logger.Warn("Failed to render presence component",
			"error", err,
			"attempt", attempt,
			"max_attempts", 3)

		if attempt < 3 {
			// Brief backoff before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 100 * time.Millisecond):
			}
		}
	}

	if err != nil {
		ps.logger.Error("Failed to render presence component after retries", "error", err)
		// Don't return error - this would cause the subscriber to stop
		return nil
	}

	ps.logger.Info("Successfully rendered presence component", "html_size", len(renderedHTML))

	// Publish HTML fragment with HTMX out-of-band swap
	htmlWithOOB := `<div hx-swap-oob="innerHTML:#presence-container">` + string(renderedHTML) + `</div>`

	htmlMsg := pubsub.Message{
		Topic:   wsTopics.TopicHTMLBroadcast.Name(),
		Payload: []byte(htmlWithOOB),
	}

	// Retry publishing with backoff
	for attempt := 1; attempt <= 3; attempt++ {
		err = ps.publisher.Publish(ctx, htmlMsg)
		if err == nil {
			ps.logger.Info("Successfully published presence HTML update")
			return nil
		}

		ps.logger.Warn("Failed to publish presence HTML update",
			"error", err,
			"attempt", attempt,
			"max_attempts", 3)

		if attempt < 3 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
			}
		}
	}

	ps.logger.Error("Failed to publish presence HTML update after retries", "error", err)
	// Don't return error - this would cause the subscriber to stop
	return nil
}
