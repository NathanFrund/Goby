package wargame

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"

	"github.com/nfrund/goby/internal/pubsub"
)

// DamageEvent represents a damage event in the game for HTML rendering.

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

type DamageEvent struct {
	TargetUnit    string `json:"targetUnit"`
	DamageAmount  int    `json:"damageAmount"`
	AttackingUnit string `json:"attackingUnit"`
}

// GameStateUpdate represents a raw data update for data-only clients.
type GameStateUpdate struct {
	EventType   string `json:"eventType"`
	UnitID      string `json:"unitId"`
	NewHealth   int    `json:"newHealth"`
	DamageTaken int    `json:"damageTaken"`
}

// Engine simulates game logic and publishes updates to the appropriate hubs.
type Engine struct {
	publisher pubsub.Publisher
}

// NewEngine creates a new wargame engine instance.
func NewEngine(pub pubsub.Publisher) *Engine {
	return &Engine{publisher: pub}
}

// SimulateHit simulates a unit being hit and publishes updates to both channels.
func (e *Engine) SimulateHit() {
	slog.Info("Wargame engine: Simulating a hit event.")

	// --- Generate random event data ---
	target := targetUnits[rand.Intn(len(targetUnits))]
	attacker := attackingUnits[rand.Intn(len(attackingUnits))]
	damage := rand.Intn(30) + 5 // Random damage between 5 and 34
	newHealth := 100 - damage

	// 1. Publish a structured event for the HTML-rendering subscriber.
	// This event contains all the data needed to render the HTML component.
	damageEvent := DamageEvent{TargetUnit: target.Name, DamageAmount: damage, AttackingUnit: attacker}
	if payload, err := json.Marshal(damageEvent); err == nil {
		msg := pubsub.Message{Topic: "wargame.events.damage", Payload: payload}
		e.publisher.Publish(context.Background(), msg)
		slog.Info("Wargame engine: Published 'wargame.events.damage' message.")
	} else {
		slog.Error("Wargame engine: Failed to marshal damage event", "error", err)
	}

	// 2. Publish the raw data structure for data-only clients (e.g., Game State Monitor).
	dataEvent := GameStateUpdate{EventType: "damage", UnitID: target.ID, NewHealth: newHealth, DamageTaken: damage}
	if jsonData, err := json.Marshal(dataEvent); err == nil {
		msg := pubsub.Message{Topic: "wargame.state.update", Payload: jsonData}
		e.publisher.Publish(context.Background(), msg)
		slog.Info("Wargame engine: Published 'wargame.state.update' message.")
	} else {
		slog.Error("Wargame engine: Failed to marshal JSON data", "error", err)
	}
}
