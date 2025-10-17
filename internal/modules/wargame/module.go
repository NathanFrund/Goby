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
)

// WargameModule implements the module.Module interface.
type WargameModule struct {
	module.BaseModule
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber
	renderer   rendering.Renderer
	engine     *Engine
}

// Dependencies holds all the services that the WargameModule requires to operate.
// This struct is used for constructor injection to make dependencies explicit.
type Dependencies struct {
	Publisher  pubsub.Publisher
	Subscriber pubsub.Subscriber
	Renderer   rendering.Renderer
}

// New creates a new instance of the WargameModule, injecting its dependencies.
func New(deps Dependencies) *WargameModule {
	return &WargameModule{
		publisher:  deps.Publisher,
		subscriber: deps.Subscriber,
		renderer:   deps.Renderer,
	}
}

// Name returns the unique name for the module.
func (m *WargameModule) Name() string {
	return "wargame"
}

// Shutdown is called on application termination.
func (m *WargameModule) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down WargameModule...")
	return nil
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
func (m *WargameModule) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	// Create and start the subscriber in a goroutine.
	wargameSubscriber := NewSubscriber(m.subscriber, m.publisher, m.renderer)
	go wargameSubscriber.Start(ctx)

	// --- Register HTTP Handlers ---
	slog.Info("Booting WargameModule: Setting up routes...")

	// The server mounts us under /app/wargame, so we use relative paths
	g.GET("/debug/hit", func(c echo.Context) error {
		// Pass the request's context to the background task.
		go m.engine.SimulateHit(c.Request().Context())
		return c.String(http.StatusOK, "Hit event triggered.")
	})
	return nil
}
