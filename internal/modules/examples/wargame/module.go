package wargame

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/examples/wargame/scripts"
	"github.com/nfrund/goby/internal/modules/examples/wargame/topics"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/script"
	"github.com/nfrund/goby/internal/topicmgr"
)

// KeyGameEngine is the type-safe key for accessing the wargame engine service.
var KeyGameEngine = registry.Key[*Engine]("wargame.Engine")

type WargameModule struct {
	module.BaseModule
	publisher    pubsub.Publisher
	subscriber   pubsub.Subscriber
	renderer     rendering.Renderer
	topicMgr     *topicmgr.Manager
	scriptEngine script.ScriptEngine
	scriptHelper *script.ModuleScriptHelper
	engine       *Engine
}

type Dependencies struct {
	Publisher    pubsub.Publisher
	Subscriber   pubsub.Subscriber
	Renderer     rendering.Renderer
	TopicMgr     *topicmgr.Manager
	ScriptEngine script.ScriptEngine
}

// WargameScriptProvider implements the EmbeddedScriptProvider interface
type WargameScriptProvider struct{}

func (w *WargameScriptProvider) GetEmbeddedScripts() map[string]string {
	return scripts.GetEmbeddedScripts()
}

func (w *WargameScriptProvider) GetModuleName() string {
	return "wargame"
}

func New(deps Dependencies) *WargameModule {
	// Create script configuration
	scriptConfig := &script.ModuleScriptConfig{
		MessageHandlers: map[string]string{
			topics.TopicEventDamage.Name():  "event_processor",
			topics.TopicStateUpdate.Name():  "event_processor",
			topics.TopicPlayerAction.Name(): "event_processor",
		},
		EndpointScripts: map[string]string{
			"/debug/hit": "hit_simulator",
		},
		DefaultLimits: script.GetDefaultSecurityLimits(),
		AutoExtract:   false, // Don't auto-extract by default
	}

	// Create script helper and register embedded scripts
	var scriptHelper *script.ModuleScriptHelper
	if deps.ScriptEngine != nil {
		scriptHelper = script.NewModuleScriptHelper(deps.ScriptEngine, "wargame", scriptConfig)

		// Register embedded scripts immediately
		provider := &WargameScriptProvider{}
		scriptHelper.RegisterEmbeddedScripts(provider)
		slog.Info("Registered wargame embedded scripts during module creation")
	}

	return &WargameModule{
		publisher:    deps.Publisher,
		subscriber:   deps.Subscriber,
		renderer:     deps.Renderer,
		topicMgr:     deps.TopicMgr,
		scriptEngine: deps.ScriptEngine,
		scriptHelper: scriptHelper,
	}
}

func (m *WargameModule) Name() string {
	return "wargame"
}

// GetScriptConfig returns script configuration for this module
func (m *WargameModule) GetScriptConfig() *script.ModuleScriptConfig {
	return &script.ModuleScriptConfig{
		MessageHandlers: map[string]string{
			topics.TopicEventDamage.Name():  "event_processor",
			topics.TopicStateUpdate.Name():  "event_processor",
			topics.TopicPlayerAction.Name(): "event_processor",
		},
		EndpointScripts: map[string]string{
			"/debug/hit": "hit_simulator",
		},
		DefaultLimits: script.GetDefaultSecurityLimits(),
		AutoExtract:   false,
	}
}

// GetExposedFunctions returns functions available to scripts
func (m *WargameModule) GetExposedFunctions() map[string]interface{} {
	return map[string]interface{}{
		// Damage calculation function
		"calculate_damage": func(weaponType string, baseDamage int, targetUnit string, attacker string) map[string]interface{} {
			// This will be called by scripts to calculate damage
			return map[string]interface{}{
				"weapon_type": weaponType,
				"base_damage": baseDamage,
				"target_unit": targetUnit,
				"attacker":    attacker,
			}
		},

		// Publish event function
		"publish_event": func(eventType string, eventData map[string]interface{}) error {
			// Allow scripts to publish events back to the system
			if m.publisher == nil {
				return fmt.Errorf("publisher not available")
			}

			// This is a simplified version - in practice you'd want more validation
			slog.Debug("Script publishing event", "type", eventType, "data", eventData)
			return nil
		},

		// Get game state function
		"get_game_state": func() map[string]interface{} {
			// Return current game state information
			return map[string]interface{}{
				"current_turn": "player-1",
				"game_phase":   "battle",
				"active_units": 3,
			}
		},

		// Random number generator (seeded)
		"rand": func() float64 {
			// Provide a random number generator to scripts
			return 0.5 // Simplified for now
		},
	}
}

func (m *WargameModule) Register(reg *registry.Registry) error {
	slog.Info("Initializing wargame engine")

	// Register all wargame topics
	if err := topics.RegisterTopics(); err != nil {
		return fmt.Errorf("failed to register wargame topics: %w", err)
	}

	// Scripts are already registered during module creation

	m.engine = NewEngine(m.publisher, m.topicMgr)
	registry.Set(reg, KeyGameEngine, m.engine)
	return nil
}

func (m *WargameModule) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	// Create and start the subscriber in a goroutine with script support
	var scriptExecutor *script.ScriptExecutor
	if m.scriptHelper != nil {
		scriptExecutor = m.scriptHelper.GetExecutor()
	}

	wargameSubscriber := NewSubscriber(m.subscriber, m.publisher, m.renderer, scriptExecutor, m.GetExposedFunctions())
	go wargameSubscriber.Start(ctx)

	// Register HTTP handlers with script integration
	g.GET("/debug/hit", func(c echo.Context) error {
		// Execute hit simulator script if available
		if m.scriptHelper != nil {
			executor := m.scriptHelper.GetExecutor()

			// Prepare HTTP request data for script
			httpRequest := &script.HTTPRequestData{
				Method:  c.Request().Method,
				Path:    c.Request().URL.Path,
				Headers: make(map[string]string),
				Body:    []byte{},
				Query:   make(map[string]string),
			}

			// Copy headers
			for key, values := range c.Request().Header {
				if len(values) > 0 {
					httpRequest.Headers[key] = values[0]
				}
			}

			// Copy query parameters
			for key, values := range c.Request().URL.Query() {
				if len(values) > 0 {
					httpRequest.Query[key] = values[0]
				}
			}

			// Execute the script
			output, err := executor.ExecuteEndpointScript(
				c.Request().Context(),
				"/debug/hit",
				httpRequest,
				m.GetExposedFunctions(),
			)

			if err != nil {
				slog.Error("Script execution failed for hit endpoint", "error", err)
				// Fall back to original behavior
				go m.engine.SimulateHit(c.Request().Context())
				return c.String(http.StatusOK, "Hit event triggered (script failed, used fallback).")
			}

			if output != nil {
				slog.Info("Hit simulator script executed successfully",
					"execution_time", output.Metrics.ExecutionTime,
					"result_type", fmt.Sprintf("%T", output.Result))

				// You could use the script result to influence the simulation
				// For now, still trigger the original simulation
				go m.engine.SimulateHit(c.Request().Context())

				return c.JSON(http.StatusOK, map[string]interface{}{
					"message":        "Hit event triggered with script enhancement",
					"script_result":  output.Result,
					"execution_time": output.Metrics.ExecutionTime.String(),
				})
			}
		}

		// Fallback to original behavior if no script helper
		go m.engine.SimulateHit(c.Request().Context())
		return c.String(http.StatusOK, "Hit event triggered.")
	})

	return nil
}

func (m *WargameModule) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down WargameModule...")
	return nil
}
