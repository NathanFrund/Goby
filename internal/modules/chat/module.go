package chat

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/hub"
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
	// 1. Retrieve the real-time communication hub
	hubVal, ok := sl.Get(string(registry.HTMLHubKey))
	if !ok {
		// Log an error if the hub dependency is not met, as it's critical for this module.
		return fmt.Errorf("HTML hub (Key: %s) not found in service locator", registry.HTMLHubKey)
	}
	h := hubVal.(*hub.Hub)

	// 2. Retrieve the template renderer
	rendererVal, ok := sl.Get(string(registry.TemplateRendererKey))
	if !ok {
		return fmt.Errorf("template renderer (Key: %s) not found in service locator", registry.TemplateRendererKey)
	}
	r := rendererVal.(rendering.Renderer)

	// 3. Instantiate the Handler, injecting its dependencies.
	handler := NewHandler(h, r)
	sl.Set(string(registry.ChatHandlerKey), handler)

	return nil
}

// Boot sets up the routes for the chat module.
func (m *ChatModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
	// --- Start Background Services ---
	// The Boot method is the ideal place to start any background workers
	// that a module requires, such as this pub/sub subscriber.

	// Retrieve dependencies for the subscriber.
	pubSubVal, _ := sl.Get(string(registry.PubSubKey))
	bridgeVal, _ := sl.Get(string(registry.WebsocketBridgeKey))
	rendererVal, _ := sl.Get(string(registry.TemplateRendererKey))

	// Type-assert the dependencies.
	sub, ok1 := pubSubVal.(pubsub.Subscriber)
	bridge, ok2 := bridgeVal.(*websocket.WebsocketBridge)
	renderer, ok3 := rendererVal.(rendering.Renderer)

	if !ok1 || !ok2 || !ok3 {
		return fmt.Errorf("chat module subscriber could not resolve dependencies")
	}

	// Create and start the subscriber in a goroutine.
	chatSubscriber := NewChatSubscriber(sub, bridge, renderer)
	go chatSubscriber.Start(context.Background()) // Using a cancellable context is a good future improvement.

	// --- Register HTTP Handlers ---
	slog.Info("Booting ChatModule: Setting up routes")
	handlerVal, ok := sl.Get(string(registry.ChatHandlerKey))
	if !ok {
		return fmt.Errorf("chat handler (Key: %s) not found in service locator", registry.ChatHandlerKey)
	}
	handler := handlerVal.(*Handler)

	// Set up routes
	g.GET("/chat", handler.ChatGet)
	// The /ws/html endpoint is now handled globally by the new V2 websocket.Bridge
	// in server/routes.go as part of the strangler fig migration.
	// g.GET("/ws/html", handler.ServeWS)

	return nil
}
