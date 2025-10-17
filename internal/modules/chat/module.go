package chat

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
)

// ChatModule implements the module.Module interface for the chat feature.
type ChatModule struct {
	module.BaseModule
	publisher  pubsub.Publisher
	subscriber pubsub.Subscriber
	renderer   rendering.Renderer
}

// Dependencies holds all the services that the ChatModule requires to operate.
// This struct is used for constructor injection to make dependencies explicit.
type Dependencies struct {
	Publisher  pubsub.Publisher
	Subscriber pubsub.Subscriber
	Renderer   rendering.Renderer
}

// New creates a new instance of the ChatModule, injecting its dependencies.
func New(deps Dependencies) *ChatModule {
	return &ChatModule{
		publisher:  deps.Publisher,
		subscriber: deps.Subscriber,
		renderer:   deps.Renderer,
	}
}

// Name returns the module name.
func (m *ChatModule) Name() string {
	return "chat"
}

// Shutdown is called on application termination.
func (m *ChatModule) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down ChatModule...")
	// In a real module, you might wait for background workers to finish here.
	return nil
}

// Boot sets up the routes and starts background services for the chat module.
func (m *ChatModule) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
	// --- Start Background Services ---

	// Create and start the subscriber in a goroutine.
	// Dependencies are now injected via the constructor and stored on the module.
	chatSubscriber := NewChatSubscriber(m.subscriber, m.publisher, m.renderer)
	go chatSubscriber.Start(ctx)

	// --- Register HTTP Handlers ---
	slog.Info("Booting ChatModule: Setting up routes...")
	handler := NewHandler(m.publisher)

	// Set up routes - the server mounts us under /app/chat, so we use root paths here
	g.GET("", handler.ChatGet)
	g.POST("/message", handler.MessagePost)

	return nil
}
