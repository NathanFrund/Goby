package wargame

import (
	"bytes"
	"embed"
	"encoding/json"
	"log/slog"
	"math/rand/v2"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/templates"
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
	renderer echo.Renderer
}

// NewEngine creates a new wargame engine instance.
func NewEngine(htmlHub, dataHub *hub.Hub, r echo.Renderer) *Engine {
	return &Engine{htmlHub: htmlHub, dataHub: dataHub, renderer: r}
}

//go:embed templates/components/*.html
var templatesFS embed.FS

// RegisterTemplates registers embedded templates for the wargame module under the "wargame" namespace.
func RegisterTemplates(r *templates.Renderer) {
	if err := r.AddStandaloneFromFS(templatesFS, "templates/components", "wargame"); err != nil {
		slog.Error("Failed to register wargame embedded components", "error", err)
	}
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
	htmlEvent := DamageEvent{TargetUnit: target.Name, DamageAmount: damage, AttackingUnit: attacker}
	var buf bytes.Buffer
	if err := e.renderer.Render(&buf, "wargame/wargame-damage.html", htmlEvent, nil); err == nil {
		e.htmlHub.Broadcast <- buf.Bytes()
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
