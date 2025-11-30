package wargame

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nfrund/goby/internal/modules/wargame/components"
	"github.com/nfrund/goby/internal/modules/wargame/events"
	"github.com/nfrund/goby/internal/modules/wargame/topics"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/script"
)

type Subscriber struct {
	subscriber     pubsub.Subscriber
	publisher      pubsub.Publisher
	renderer       rendering.Renderer
	scriptExecutor *script.ScriptExecutor
	exposedFuncs   map[string]interface{}
}

func NewSubscriber(sub pubsub.Subscriber, pub pubsub.Publisher, renderer rendering.Renderer, scriptExecutor *script.ScriptExecutor, exposedFuncs map[string]interface{}) *Subscriber {
	return &Subscriber{
		subscriber:     sub,
		publisher:      pub,
		renderer:       renderer,
		scriptExecutor: scriptExecutor,
		exposedFuncs:   exposedFuncs,
	}
}

func (s *Subscriber) Start(ctx context.Context) {
	slog.Info("Starting wargame module subscriber")

	// Listen for HTML events
	go func() {
		err := pubsub.Subscribe(ctx, s.subscriber, topics.TopicEventDamage, s.handleDamageEvent)
		if err != nil && err != context.Canceled {
			slog.Error("Wargame HTML subscriber stopped with error", "error", err)
		}
	}()

	// Listen for Data events
	go func() {
		err := pubsub.Subscribe(ctx, s.subscriber, topics.TopicStateUpdate, s.handleStateUpdateEvent)
		if err != nil && err != context.Canceled {
			slog.Error("Wargame Data subscriber stopped with error", "error", err)
		}
	}()

	// Listen for player actions
	go func() {
		err := pubsub.Subscribe(ctx, s.subscriber, topics.TopicPlayerAction, s.handlePlayerAction)
		if err != nil && err != context.Canceled {
			slog.Error("Wargame Actions subscriber stopped with error", "error", err)
		}
	}()
}

func (s *Subscriber) handleDamageEvent(ctx context.Context, event events.Damage) error {
	// Execute event processor script if available
	if s.scriptExecutor != nil {
		// Re-marshal for script execution (scripts expect pubsub.Message)
		data, _ := json.Marshal(event)
		msg := &pubsub.Message{
			Topic:   topics.TopicEventDamage.Name(),
			Payload: data,
		}
		output, err := s.scriptExecutor.ExecuteMessageHandler(ctx, msg.Topic, msg, s.exposedFuncs)
		if err != nil {
			slog.Error("Script execution failed for damage event", "error", err)
			// Continue with original processing even if script fails
		} else if output != nil {
			slog.Info("Event processor script executed for damage event",
				"execution_time", output.Metrics.ExecutionTime,
				"chain_reaction", output.Result)

			// Check if script indicates chain reaction
			if result, ok := output.Result.(map[string]interface{}); ok {
				if chainReaction, exists := result["chain_reaction"]; exists && chainReaction == true {
					slog.Info("Chain reaction detected by script", "follow_up_events", result["follow_up_events"])
					// Here you could process follow-up events if needed
				}
			}
		}
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

func (s *Subscriber) handleStateUpdateEvent(ctx context.Context, event events.StateUpdate) error {
	// Execute event processor script if available
	if s.scriptExecutor != nil {
		// Re-marshal for script execution
		data, _ := json.Marshal(event)
		msg := &pubsub.Message{
			Topic:   topics.TopicStateUpdate.Name(),
			Payload: data,
		}
		output, err := s.scriptExecutor.ExecuteMessageHandler(ctx, msg.Topic, msg, s.exposedFuncs)
		if err != nil {
			slog.Error("Script execution failed for state update event", "error", err)
		} else if output != nil {
			slog.Info("Event processor script executed for state update",
				"execution_time", output.Metrics.ExecutionTime,
				"active_units", output.Result)
		}
	}

	// Re-marshal the event for forwarding
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Forward the raw state to data clients
	return s.publisher.Publish(ctx, pubsub.Message{
		Topic:   "ws.data.broadcast", // Will be updated when WebSocket integration is complete
		Payload: payload,
	})
}

func (s *Subscriber) handlePlayerAction(ctx context.Context, action events.PlayerAction) error {
	slog.InfoContext(ctx, "Processing player action",
		"player_id", action.PlayerID,
		"action", action.Action,
	)

	// Execute event processor script if available
	if s.scriptExecutor != nil {
		// Re-marshal for script execution
		data, _ := json.Marshal(action)
		msg := &pubsub.Message{
			Topic:   topics.TopicPlayerAction.Name(),
			Payload: data,
		}
		output, err := s.scriptExecutor.ExecuteMessageHandler(ctx, msg.Topic, msg, s.exposedFuncs)
		if err != nil {
			slog.Error("Script execution failed for player action", "error", err)
		} else if output != nil {
			slog.Info("Event processor script executed for player action",
				"execution_time", output.Metrics.ExecutionTime,
				"requires_validation", output.Result)

			// Check if script indicates validation is required
			if result, ok := output.Result.(map[string]interface{}); ok {
				if requiresValidation, exists := result["requires_validation"]; exists && requiresValidation == true {
					slog.Info("Player action requires validation", "action", action.Action, "player", action.PlayerID)
					// Here you could add validation logic
				}
			}
		}
	}

	// Process the action here
	return nil
}
