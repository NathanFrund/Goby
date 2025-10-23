package wargame

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/topicmgr"
)

var (
	targetUnits = []struct {
		ID   string
		Name string
	}{
		{ID: "alpha-squad", Name: "Alpha Squad"},
		{ID: "bravo-team", Name: "Bravo Team"},
		{ID: "command-post", Name: "Command Post"},
	}
	attackingUnits = []string{"Enemy Sniper", "Artillery Strike", "Tank", "Ambush"}
)

type Engine struct {
	publisher pubsub.Publisher
	topicMgr  *topicmgr.Manager
}

func NewEngine(pub pubsub.Publisher, topicMgr *topicmgr.Manager) *Engine {
	return &Engine{
		publisher: pub,
		topicMgr:  topicMgr,
	}
}

func (e *Engine) SimulateHit(ctx context.Context) {
	logger := middleware.FromContext(ctx)

	// Generate random event data
	target := targetUnits[rand.Intn(len(targetUnits))]
	attacker := attackingUnits[rand.Intn(len(attackingUnits))]

	// Use default damage for now - this could be enhanced to use script-based calculation
	damage := rand.Intn(30) + 5

	// Create and publish damage event
	event := DamageEvent{
		BaseMessage: BaseMessage{
			MessageID: uuid.New().String(),
			Timestamp: time.Now().UTC(),
			Version:   "1.0",
		},
		TargetUnit:   target.Name,
		DamageAmount: damage,
		Attacker:     attacker,
		WeaponType:   "ballistic",
	}

	if err := e.publishEvent(ctx, EventDamage, event); err != nil {
		logger.Error("Failed to publish damage event", "error", err)
	}

	// Update and publish game state
	state := e.calculateNewState(event)
	if err := e.publishEvent(ctx, StateUpdate, state); err != nil {
		logger.Error("Failed to publish state update", "error", err)
	}
}

func (e *Engine) publishEvent(ctx context.Context, topic topicmgr.Topic, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return e.publisher.Publish(ctx, pubsub.Message{
		Topic:   topic.Name(),
		Payload: data,
	})
}

func (e *Engine) calculateNewState(event DamageEvent) GameState {
	// Simplified for example - implement actual game state logic here
	return GameState{
		BaseMessage: BaseMessage{
			MessageID: uuid.New().String(),
			Timestamp: time.Now().UTC(),
			Version:   "1.0",
		},
		Units: []UnitState{
			{
				ID:        "unit-1",
				Name:      event.TargetUnit,
				Health:    100 - event.DamageAmount,
				MaxHealth: 100,
				Position:  "A1",
				Status:    "active",
			},
		},
		CurrentTurn: "player-1",
		GamePhase:   "battle",
		LastUpdated: time.Now().UTC(),
	}
}
