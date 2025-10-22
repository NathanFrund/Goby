package wargame

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nfrund/goby/internal/modules/wargame/components"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
)

type Subscriber struct {
	subscriber pubsub.Subscriber
	publisher  pubsub.Publisher
	renderer   rendering.Renderer
}

func NewSubscriber(sub pubsub.Subscriber, pub pubsub.Publisher, renderer rendering.Renderer) *Subscriber {
	return &Subscriber{
		subscriber: sub,
		publisher:  pub,
		renderer:   renderer,
	}
}

func (s *Subscriber) Start(ctx context.Context) {
	slog.Info("Starting wargame module subscriber")

	// Listen for HTML events
	go func() {
		err := s.subscriber.Subscribe(ctx, EventDamage.Name(), s.handleDamageEvent)
		if err != nil && err != context.Canceled {
			slog.Error("Wargame HTML subscriber stopped with error", "error", err)
		}
	}()

	// Listen for Data events
	go func() {
		err := s.subscriber.Subscribe(ctx, StateUpdate.Name(), s.handleStateUpdateEvent)
		if err != nil && err != context.Canceled {
			slog.Error("Wargame Data subscriber stopped with error", "error", err)
		}
	}()

	// Listen for player actions
	go func() {
		err := s.subscriber.Subscribe(ctx, PlayerAction.Name(), s.handlePlayerAction)
		if err != nil && err != context.Canceled {
			slog.Error("Wargame Actions subscriber stopped with error", "error", err)
		}
	}()
}

func (s *Subscriber) handleDamageEvent(ctx context.Context, msg pubsub.Message) error {
	var event DamageEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		slog.Error("Failed to unmarshal wargame damage event payload", "error", err)
		return err
	}

	// Generate a unique message ID for the HTML component
	messageID := "damage-" + uuid.New().String()

	// 1. Send HTML version to chat
	component := components.DamageEvent(event.TargetUnit, event.DamageAmount, event.Attacker, messageID)
	renderedHTML, err := s.renderer.RenderComponent(ctx, component)
	if err != nil {
		slog.Error("Failed to render wargame damage component", "error", err)
		return err
	}

	// 2. Send JSON version to game state monitor
	jsonData, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal game state update", "error", err)
		return err
	}

	// Send both messages using typed topics
	if err := s.publisher.Publish(ctx, pubsub.Message{
		Topic:   "ws.html.broadcast", // Will be updated when WebSocket integration is complete
		Payload: renderedHTML,
	}); err != nil {
		return err
	}

	return s.publisher.Publish(ctx, pubsub.Message{
		Topic:   "ws.data.broadcast", // Will be updated when WebSocket integration is complete
		Payload: jsonData,
	})
}

func (s *Subscriber) handleStateUpdateEvent(ctx context.Context, msg pubsub.Message) error {
	// Forward the raw state to data clients
	return s.publisher.Publish(ctx, pubsub.Message{
		Topic:   "ws.data.broadcast", // Will be updated when WebSocket integration is complete
		Payload: msg.Payload,
	})
}

func (s *Subscriber) handlePlayerAction(ctx context.Context, msg pubsub.Message) error {
	var action Action
	if err := json.Unmarshal(msg.Payload, &action); err != nil {
		return fmt.Errorf("failed to unmarshal player action: %w", err)
	}

	slog.InfoContext(ctx, "Processing player action",
		"player_id", action.PlayerID,
		"action", action.Action,
	)

	// Process the action here
	return nil
}
