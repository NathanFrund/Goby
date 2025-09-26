package chat

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/templates"
)

// ChatModule implements the module.Module interface for the chat feature.
type ChatModule struct{}

// Name returns the module name.
func (m *ChatModule) Name() string {
	return "chat"
}

// RegisterTemplates is a placeholder for template registration.
// The chat module currently uses shared templates.
func (m *ChatModule) RegisterTemplates(renderer *templates.Renderer) {
	// No module-specific templates to register for chat at the moment.
}

// Register binds the chat handler into the service container.
func (m *ChatModule) Register(sl registry.ServiceLocator, cfg config.Provider) error {
	htmlHubVal, ok := sl.Get(string(registry.HTMLHubKey))
	if !ok {
		return fmt.Errorf("HTML hub not found in service locator")
	}
	htmlHub := htmlHubVal.(*hub.Hub)

	rendererVal, ok := sl.Get(string(registry.TemplateRendererKey))
	if !ok {
		return fmt.Errorf("template renderer not found in service locator")
	}
	renderer := rendererVal.(*templates.Renderer)

	handler := NewHandler(htmlHub, renderer)
	sl.Set(string(registry.ChatHandlerKey), handler)

	return nil
}

// Boot sets up the routes for the chat module.
func (m *ChatModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
	handlerVal, ok := sl.Get(string(registry.ChatHandlerKey))
	if !ok {
		return fmt.Errorf("chat handler not found in service locator")
	}
	handler := handlerVal.(*Handler)

	userStoreVal, ok := sl.Get(string(registry.UserStoreKey))
	if !ok {
		return fmt.Errorf("user store not found in service locator")
	}
	userStore := userStoreVal.(domain.UserRepository)
	authMiddleware := middleware.Auth(userStore)

	// Register chat routes, which are already protected by the group's auth middleware.
	g.GET("/chat", handler.ChatGet, authMiddleware)
	g.GET("/ws/html", handler.ServeWS, authMiddleware)

	return nil
}
