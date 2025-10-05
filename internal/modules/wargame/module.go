package wargame

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/websocket"
)

// WargameModule implements the module.Module interface.
type WargameModule struct{}

// Name returns the unique name for the module.
func (m *WargameModule) Name() string {
	return "wargame"
}

// Register creates the Wargame Engine and registers it in the service locator.
func (m *WargameModule) Register(sl registry.ServiceLocator, cfg config.Provider) error {
	pubSubVal, ok := sl.Get(string(registry.PubSubKey))
	if !ok {
		return fmt.Errorf("pub/sub service not found for wargame module")
	}
	publisher := pubSubVal.(pubsub.Publisher)

	slog.Info("Initializing wargame engine")
	wargameEngine := NewEngine(publisher)
	sl.Set(string(registry.WargameEngineKey), wargameEngine)

	return nil
}

// Boot registers the HTTP routes for the wargame module.
func (m *WargameModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
	// --- Start Background Services ---
	pubSubVal, _ := sl.Get(string(registry.PubSubKey))
	bridgeVal, _ := sl.Get(string(registry.NewWebsocketBridgeKey))
	rendererVal, _ := sl.Get(string(registry.TemplateRendererKey))

	sub, ok1 := pubSubVal.(pubsub.Subscriber)
	bridge, ok2 := bridgeVal.(*websocket.Bridge)
	renderer, ok3 := rendererVal.(rendering.Renderer)

	if !ok1 || !ok2 || !ok3 || bridge == nil {
		return fmt.Errorf("wargame module subscriber could not resolve dependencies")
	}

	// Create and start the subscriber in a goroutine.
	wargameSubscriber := NewSubscriber(sub, bridge, renderer)
	go wargameSubscriber.Start(context.Background())

	// --- Register HTTP Handlers ---
	slog.Info("Booting WargameModule: Setting up routes")

	// This logic is moved from the old init-based route registration.
	wargameEngineVal, ok := sl.Get(string(registry.WargameEngineKey))
	if !ok {
		slog.Warn("Wargame engine not found in service locator, skipping route registration.")
		return nil
	}
	wargameEngine := wargameEngineVal.(*Engine)

	g.GET("/debug/hit", func(c echo.Context) error {
		go wargameEngine.SimulateHit()
		return c.String(http.StatusOK, "Hit event triggered.")
	})
	return nil
}
