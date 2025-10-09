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
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber
	bridge     websocket.Bridge
	renderer   rendering.Renderer
	engine     *Engine
}

// Dependencies holds all the services that the WargameModule requires to operate.
// This struct is used for constructor injection to make dependencies explicit.
type Dependencies struct {
	Publisher  pubsub.Publisher
	Subscriber pubsub.Subscriber
	Bridge     websocket.Bridge
	Renderer   rendering.Renderer
}

// New creates a new instance of the WargameModule, injecting its dependencies.
func New(deps Dependencies) *WargameModule {
	return &WargameModule{
		publisher:  deps.Publisher,
		subscriber: deps.Subscriber,
		bridge:     deps.Bridge,
		renderer:   deps.Renderer,
	}
}

// Name returns the unique name for the module.
func (m *WargameModule) Name() string {
	return "wargame"
}

// Register creates the Wargame Engine and registers it in the service locator.
func (m *WargameModule) Register(reg *registry.Registry) error {
	slog.Info("Initializing wargame engine")
	m.engine = NewEngine(m.publisher)

	// Register the concrete *Engine type so other modules could use it if needed.
	// The wargame module itself will use its internal reference.
	reg.Set((**Engine)(nil), m.engine)

	return nil
}

// Boot registers the HTTP routes for the wargame module.
func (m *WargameModule) Boot(g *echo.Group, reg *registry.Registry) error {
	// Create and start the subscriber in a goroutine.
	wargameSubscriber := NewSubscriber(m.subscriber, m.bridge, m.renderer)
	go wargameSubscriber.Start(context.Background())

	// --- Register HTTP Handlers ---
	slog.Info("Booting WargameModule: Setting up routes...")

	// The server mounts us under /app/wargame, so we use relative paths
	g.GET("/debug/hit", func(c echo.Context) error {
		go m.engine.SimulateHit()
		return c.String(http.StatusOK, "Hit event triggered.")
	})
	return nil
}
