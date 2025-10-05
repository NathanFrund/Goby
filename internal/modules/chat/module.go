package chat

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/websocket"
)

// ChatModule implements the module.Module interface for the chat feature.
type ChatModule struct{}

// Name returns the module name.
func (m *ChatModule) Name() string {
	return "chat"
}

// TemplateFS is removed: Component libraries (like templ/gomponents) are compiled
// Go functions, not loaded from an embedded filesystem (fs.FS).

// Register binds the chat handler into the service container.
func (m *ChatModule) Register(sl registry.ServiceLocator, cfg config.Provider) error {
	// The handler is now instantiated directly in Boot, so there is nothing
	// to register here for the chat module.
	return nil
}

// Boot sets up the routes for the chat module.
func (m *ChatModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
	// --- Start Background Services ---
	// The Boot method is the ideal place to start any background workers
	// that a module requires, such as this pub/sub subscriber.

	// Retrieve dependencies for the subscriber.
	pubSubVal, _ := sl.Get(string(registry.PubSubKey))
	bridgeVal, _ := sl.Get(string(registry.NewWebsocketBridgeKey))
	rendererVal, _ := sl.Get(string(registry.TemplateRendererKey))

	// Type-assert the dependencies.
	sub, ok1 := pubSubVal.(pubsub.Subscriber)
	bridge, ok2 := bridgeVal.(*websocket.Bridge)
	renderer, ok3 := rendererVal.(rendering.Renderer)

	if !ok1 || !ok2 || !ok3 || bridge == nil {
		return fmt.Errorf("chat module subscriber could not resolve dependencies")
	}

	// Create and start the subscriber in a goroutine.
	chatSubscriber := NewChatSubscriber(sub, bridge, renderer)
	go chatSubscriber.Start(context.Background()) // Using a cancellable context is a good future improvement.

	// --- Register HTTP Handlers ---
	slog.Info("Booting ChatModule: Setting up routes")

	// The handler now depends on the publisher, so we resolve it and
	// instantiate the handler here.
	publisher := pubSubVal.(pubsub.Publisher)
	handler := NewHandler(publisher)

	// Set up routes
	g.GET("/chat", handler.ChatGet)
	g.POST("/chat/message", handler.MessagePost)

	return nil
}
