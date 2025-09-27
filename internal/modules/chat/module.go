package chat

import (
	"fmt"
	"io/fs"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/registry"
)

// ChatModule implements the module.Module interface for the new chat feature.
type ChatModule struct{}

// Name returns the module name.
func (m *ChatModule) Name() string {
	return "chat"
}

// TemplateFS returns the embedded filesystem for the module's templates.
func (m *ChatModule) TemplateFS() fs.FS {
	return templatesFS
}

// Register binds the chat2 handler into the service container.
func (m *ChatModule) Register(sl registry.ServiceLocator, cfg config.Provider) error {
	hubVal, ok := sl.Get(string(registry.HTMLHubKey))
	if !ok {
		return fmt.Errorf("HTML hub not found in service locator")
	}
	hub := hubVal.(*hub.Hub)

	rendererVal, ok := sl.Get(string(registry.TemplateRendererKey))
	if !ok {
		return fmt.Errorf("template renderer not found in service locator")
	}
	renderer := rendererVal.(echo.Renderer)

	handler := NewHandler(hub, renderer)
	sl.Set(string(registry.ChatHandlerKey), handler)

	return nil
}

// Boot sets up the routes for the chat2 module.
func (m *ChatModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
	handlerVal, ok := sl.Get(string(registry.ChatHandlerKey))
	if !ok {
		return fmt.Errorf("chat handler not found in service locator")
	}
	handler := handlerVal.(*Handler)

	// Set up routes
	g.GET("/chat", handler.ChatGet)
	g.GET("/ws/chat", handler.ServeWS)

	return nil
}
