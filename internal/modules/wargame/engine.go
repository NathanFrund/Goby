package wargame

import (
	"bytes"
	"encoding/json"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/hub"
)

// DamageEvent represents a damage event in the game for HTML rendering.
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
	renderer echo.Renderer
}

// NewEngine creates a new wargame engine instance.
func NewEngine(htmlHub, dataHub *hub.Hub, r echo.Renderer) *Engine {
	return &Engine{htmlHub: htmlHub, dataHub: dataHub, renderer: r}
}

// SimulateHit simulates a unit being hit and publishes updates to both channels.
func (e *Engine) SimulateHit() {
	slog.Info("Wargame engine: Simulating a hit event.")

	// 1. Publish the rendered HTML fragment to the HTML hub for web clients.
	htmlEvent := DamageEvent{TargetUnit: "Alpha Squad", DamageAmount: 15, AttackingUnit: "Enemy Sniper"}
	var buf bytes.Buffer
	if err := e.renderer.Render(&buf, "wargame-damage.html", htmlEvent, nil); err == nil {
		e.htmlHub.Broadcast <- buf.Bytes()
		slog.Info("Wargame engine: Published HTML fragment to htmlHub.")
	} else {
		slog.Error("Wargame engine: Failed to render HTML fragment", "error", err)
	}

	// 2. Publish the raw data structure to the data hub for non-web clients.
	dataEvent := GameStateUpdate{EventType: "damage", UnitID: "alpha-squad", NewHealth: 85, DamageTaken: 15}
	if jsonData, err := json.Marshal(dataEvent); err == nil {
		e.dataHub.Broadcast <- jsonData
		slog.Info("Wargame engine: Published JSON data to dataHub.")
	} else {
		slog.Error("Wargame engine: Failed to marshal JSON data", "error", err)
	}
}
