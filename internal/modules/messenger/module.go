package messenger

import (
	"fmt"
	"io/fs"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/registry"
)

// MessengerModule implements the module.Module interface
// and handles the messenger functionality.
type MessengerModule struct{}

// Name returns the module name
func (m *MessengerModule) Name() string {
	return "messenger"
}

// TemplateFS returns the embedded filesystem for the module's templates.
func (m *MessengerModule) TemplateFS() fs.FS {
	return nil // Stub implementation; this module has no templates for now.
}

// Register is for binding services into the service container.
func (m *MessengerModule) Register(sl registry.ServiceLocator, cfg config.Provider) error {
	// Initialize store
	store := NewStore(sl)

	// Initialize and register handler
	handler := NewHandler(store, sl)
	sl.Set(string(registry.MessengerHandlerKey), handler)

	return nil
}

// Boot is called after all modules have been registered.
// It's used to set up routes and other runtime configurations.
func (m *MessengerModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
	handlerVal, ok := sl.Get(string(registry.MessengerHandlerKey))
	if !ok {
		return fmt.Errorf("messenger handler not found in service locator")
	}
	handler := handlerVal.(Handler)

	userStoreVal, ok := sl.Get(string(registry.UserStoreKey))
	if !ok {
		return fmt.Errorf("user store not found in service locator")
	}
	userStore, ok := userStoreVal.(domain.UserRepository)
	if !ok {
		return fmt.Errorf("user store in service locator has wrong type")
	}
	authMiddleware := middleware.Auth(userStore)
	// Messenger UI (prefixed with /app by the server)
	g.GET("/messenger", handler.ChatUI, authMiddleware)

	// API endpoints (prefixed with /api by the server)
	api := g.Group("/messenger")
	{
		api.POST("/messages", handler.CreateMessage, authMiddleware)
		api.GET("/messages", handler.ListMessages, authMiddleware)
	}

	// WebSocket endpoint (prefixed with /ws by the server)
	g.GET("/messenger", handler.WebSocket, authMiddleware)
	return nil
}
