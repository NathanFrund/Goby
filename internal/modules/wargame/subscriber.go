package wargame

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nfrund/goby/internal/modules/wargame/templates/components"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/websocket"
)

// Subscriber listens for wargame events on the pub/sub bus,
// renders them to HTML, and broadcasts them to HTML clients.
type Subscriber struct {
	subscriber pubsub.Subscriber
	bridge     *websocket.Bridge
	renderer   rendering.Renderer
}

// NewSubscriber creates a new subscriber for the wargame module.
func NewSubscriber(sub pubsub.Subscriber, bridge *websocket.Bridge, renderer rendering.Renderer) *Subscriber {
	return &Subscriber{
		subscriber: sub,
		bridge:     bridge,
		renderer:   renderer,
	}
}

// Start begins listening for messages on the "wargame.events.damage" topic.
func (s *Subscriber) Start(ctx context.Context) {
	slog.Info("Starting wargame module subscriber")

	// Listen for HTML events in one goroutine
	go func() {
		err := s.subscriber.Subscribe(ctx, "wargame.events.damage", s.handleDamageEvent)
		if err != nil && err != context.Canceled {
			slog.Error("Wargame HTML subscriber stopped with error", "error", err)
		}
	}()

	// Listen for Data events in another goroutine
	go func() {
		err := s.subscriber.Subscribe(ctx, "wargame.state.update", s.handleStateUpdateEvent)
		if err != nil && err != context.Canceled {
			slog.Error("Wargame Data subscriber stopped with error", "error", err)
		}
	}()
}

// handleDamageEvent processes damage events, renders them to HTML, and broadcasts.
func (s *Subscriber) handleDamageEvent(ctx context.Context, msg pubsub.Message) error {
	var event DamageEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		slog.Error("Failed to unmarshal wargame damage event payload", "error", err)
		return err
	}

	// Render the event to an HTML component.
	component := components.DamageEvent(event.TargetUnit, event.DamageAmount, event.AttackingUnit)
	renderedHTML, err := s.renderer.RenderComponent(ctx, component)
	if err != nil {
		slog.Error("Failed to render wargame damage component", "error", err)
		return err
	}

	// Broadcast the final HTML to all HTML clients via the new bridge.
	s.bridge.Broadcast(renderedHTML, websocket.ConnectionTypeHTML)
	return nil
}

// handleStateUpdateEvent processes raw game state updates and broadcasts them to data clients.
func (s *Subscriber) handleStateUpdateEvent(ctx context.Context, msg pubsub.Message) error {
	// The payload is already the JSON we want to send. Just broadcast it.
	s.bridge.Broadcast(msg.Payload, websocket.ConnectionTypeData)
	return nil
}
