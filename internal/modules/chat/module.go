package chat

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/websocket"
)

// ChatModule implements the module.Module interface for the chat feature.
type ChatModule struct {
	module.BaseModule
}

// New creates a new instance of the ChatModule.
func New() *ChatModule {
	return &ChatModule{}
}

// Name returns the module name.
func (m *ChatModule) Name() string {
	return "chat"
}

// Boot sets up the routes for the chat module.
func (m *ChatModule) Boot(g *echo.Group, reg *registry.Registry) error {
	// --- Start Background Services ---

	// Retrieve dependencies by their interface type. This is now type-safe.
	sub := registry.MustGet[pubsub.Subscriber](reg)
	bridge := registry.MustGet[websocket.Bridge](reg)
	renderer := registry.MustGet[rendering.Renderer](reg)

	// Create and start the subscriber in a goroutine.
	chatSubscriber := NewChatSubscriber(sub, bridge, renderer)
	go chatSubscriber.Start(context.Background()) // Using a cancellable context is a good future improvement.

	// --- Register HTTP Handlers ---
	slog.Info("Booting ChatModule: Setting up routes...")

	// The handler depends on the publisher, so we resolve it and instantiate the handler.
	publisher := registry.MustGet[pubsub.Publisher](reg)
	handler := NewHandler(publisher)

	// Set up routes - the server mounts us under /app/chat, so we use root paths here
	g.GET("", handler.ChatGet)
	g.POST("/message", handler.MessagePost)

	return nil
}
