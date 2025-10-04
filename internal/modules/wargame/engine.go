package wargame

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"

	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/modules/wargame/templates/components"
	"github.com/nfrund/goby/internal/rendering"
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
	TargetUnit    string
	DamageAmount  int
	AttackingUnit string
}

// GameStateUpdate represents a raw data update for non-HTML clients.
type GameStateUpdate struct {
	EventType   string `json:"eventType"`
	UnitID      string `json:"unitId"`
	NewHealth   int    `json:"newHealth"`
	DamageTaken int    `json:"damageTaken"`
}

// Engine simulates game logic and publishes updates to the appropriate hubs.
type Engine struct {
	htmlHub  *hub.Hub
	dataHub  *hub.Hub
	renderer rendering.Renderer
}

// NewEngine creates a new wargame engine instance.
func NewEngine(htmlHub, dataHub *hub.Hub, r rendering.Renderer) *Engine {
	return &Engine{htmlHub: htmlHub, dataHub: dataHub, renderer: r}
}

// SimulateHit simulates a unit being hit and publishes updates to both channels.
func (e *Engine) SimulateHit() {
	slog.Info("Wargame engine: Simulating a hit event.")

	// --- Generate random event data ---
	target := targetUnits[rand.N(len(targetUnits))]
	attacker := attackingUnits[rand.N(len(attackingUnits))]
	damage := rand.N(30) + 5 // Random damage between 5 and 34
	newHealth := 100 - damage

	// 1. Publish the rendered HTML fragment to the HTML hub for web clients.
	component := components.DamageEvent(target.Name, damage, attacker)
	htmlContent, err := e.renderer.RenderComponent(context.Background(), component)
	if err == nil {
		e.htmlHub.Broadcast <- htmlContent
		slog.Info("Wargame engine: Published HTML fragment to htmlHub.")
	} else {
		slog.Error("Wargame engine: Failed to render HTML fragment", "error", err)
	}

	// 2. Publish the raw data structure to the data hub for non-web clients.
	dataEvent := GameStateUpdate{EventType: "damage", UnitID: target.ID, NewHealth: newHealth, DamageTaken: damage}
	if jsonData, err := json.Marshal(dataEvent); err == nil {
		e.dataHub.Broadcast <- jsonData
		slog.Info("Wargame engine: Published JSON data to dataHub.")
	} else {
		slog.Error("Wargame engine: Failed to marshal JSON data", "error", err)
	}
}
