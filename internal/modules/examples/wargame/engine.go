package wargame

import (
	"context"
	"math/rand"
	"time"

	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/modules/examples/wargame/events"
	"github.com/nfrund/goby/internal/modules/examples/wargame/topics"
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
	event := events.Damage{
		TargetUnit:   target.Name,
		DamageAmount: damage,
		Attacker:     attacker,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	if err := pubsub.Publish(ctx, e.publisher, topics.TopicEventDamage, event); err != nil {
		logger.Error("Failed to publish damage event", "error", err)
	}

	// Update and publish game state
	state := e.calculateNewState(event)
	if err := pubsub.Publish(ctx, e.publisher, topics.TopicStateUpdate, state); err != nil {
		logger.Error("Failed to publish state update", "error", err)
	}
}

func (e *Engine) calculateNewState(event events.Damage) events.StateUpdate {
	// Simplified for example - implement actual game state logic here
	return events.StateUpdate{
		GameID: "game-1",
		Turn:   1,
		Phase:  "battle",
		Units: []map[string]interface{}{
			{
				"id":         "unit-1",
				"name":       event.TargetUnit,
				"health":     100 - event.DamageAmount,
				"max_health": 100,
				"position":   "A1",
				"status":     "active",
			},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
