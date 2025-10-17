package wargame

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nfrund/goby/internal/modules/wargame/templates/components"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
)

// Subscriber listens for wargame events on the pub/sub bus,
// renders them to HTML, and broadcasts them to HTML clients.
type Subscriber struct {
	subscriber pubsub.Subscriber
	publisher  pubsub.Publisher
	renderer   rendering.Renderer
}

// NewSubscriber creates a new subscriber for the wargame module.
func NewSubscriber(sub pubsub.Subscriber, pub pubsub.Publisher, renderer rendering.Renderer) *Subscriber {
	return &Subscriber{
		subscriber: sub,
		publisher:  pub,
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
			slog.ErrorContext(ctx, "Wargame Data subscriber stopped with error", "error", err)
		}
	}()

	// Listen for player actions
	go func() {
		err := s.subscriber.Subscribe(ctx, "wargame.actions", s.handlePlayerAction)
		if err != nil && err != context.Canceled {
			slog.ErrorContext(ctx, "Wargame Actions subscriber stopped with error", "error", err)
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
	return s.publisher.Publish(ctx, pubsub.Message{
		Topic:   "ws.html.broadcast",
		Payload: renderedHTML,
	})
}

// handleStateUpdateEvent processes raw game state updates and broadcasts them to data clients.
func (s *Subscriber) handleStateUpdateEvent(ctx context.Context, msg pubsub.Message) error {
	var update GameStateUpdate
	if err := json.Unmarshal(msg.Payload, &update); err != nil {
		return fmt.Errorf("failed to unmarshal game update: %w", err)
	}

	// Create a data message with the game state
	return s.publisher.Publish(ctx, pubsub.Message{
		Topic:   "ws.data.broadcast",
		Payload: msg.Payload, // Forward the original payload
	})
}

// PlayerAction represents an action taken by a player in the wargame
type PlayerAction struct {
	PlayerID string      `json:"player_id"`
	Action   string      `json:"action"`
	Data     interface{} `json:"data,omitempty"`
}

// handlePlayerAction processes player actions and updates the game state
func (s *Subscriber) handlePlayerAction(ctx context.Context, msg pubsub.Message) error {
	var action PlayerAction
	if err := json.Unmarshal(msg.Payload, &action); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal player action", "error", err, "payload", string(msg.Payload))
		return fmt.Errorf("failed to unmarshal player action: %w", err)
	}

	// Log the action with context
	slog.InfoContext(ctx, "Processing player action",
		"player_id", action.PlayerID,
		"action", action.Action,
	)

	// This is a placeholder. In a real app, you would process the action.
	// For example: validate the action, update game state, and broadcast updates.

	return nil
}
