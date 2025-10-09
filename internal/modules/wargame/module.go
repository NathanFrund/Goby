package wargame

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/websocket"
)

// WargameModule implements the module.Module interface.
type WargameModule struct {
	module.BaseModule
}

// New creates a new instance of the WargameModule.
func New() *WargameModule {
	return &WargameModule{}
}

// Name returns the unique name for the module.
func (m *WargameModule) Name() string {
	return "wargame"
}

// Register creates the Wargame Engine and registers it in the service locator.
func (m *WargameModule) Register(reg *registry.Registry) error {
	publisher := registry.MustGet[pubsub.Publisher](reg)

	slog.Info("Initializing wargame engine")
	wargameEngine := NewEngine(publisher)
	// Register the concrete *Engine type
	reg.Set((**Engine)(nil), wargameEngine)

	return nil
}

// Boot registers the HTTP routes for the wargame module.
func (m *WargameModule) Boot(g *echo.Group, reg *registry.Registry) error {
	// --- Start Background Services ---
	sub := registry.MustGet[pubsub.Subscriber](reg)
	bridge := registry.MustGet[websocket.Bridge](reg)
	renderer := registry.MustGet[rendering.Renderer](reg)

	// Create and start the subscriber in a goroutine.
	wargameSubscriber := NewSubscriber(sub, bridge, renderer)
	go wargameSubscriber.Start(context.Background())

	// --- Register HTTP Handlers ---
	slog.Info("Booting WargameModule: Setting up routes...")

	wargameEngine := registry.MustGet[*Engine](reg)

	// The server mounts us under /app/wargame, so we use relative paths
	g.GET("/debug/hit", func(c echo.Context) error {
		go wargameEngine.SimulateHit()
		return c.String(http.StatusOK, "Hit event triggered.")
	})
	return nil
}
