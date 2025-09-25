package fullchat

import (
	"fmt"
	"log/slog"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/templates"
	"github.com/surrealdb/surrealdb.go"
)

// FullChatModule implements the module.Module interface.
type FullChatModule struct{}

// Name returns the unique name for the module.
func (m *FullChatModule) Name() string {
	return "full-chat"
}

// RegisterTemplates is a placeholder as this module currently uses shared templates.
func (m *FullChatModule) RegisterTemplates(renderer *templates.Renderer) {
	// No module-specific templates to register at this time.
}

// Register reads the module's configuration and binds the full-chat Store
// into the service locator. The old service is no longer used.
func (m *FullChatModule) Register(sl registry.ServiceLocator, appCfg config.Provider) error {
	modCfg, exists := appCfg.GetModuleConfig(m.Name())
	if !exists {
		slog.Warn("full-chat module configuration not found, skipping service registration.")
		return nil // Don't block startup for an optional module.
	}

	fullChatCfg, ok := modCfg.(*Config)
	if !ok {
		return fmt.Errorf("invalid full-chat module configuration type")
	}

	if fullChatCfg.SurrealNS == "" || fullChatCfg.SurrealDB == "" {
		slog.Warn("Missing FULLCHAT_SURREAL_NS/FULLCHAT_SURREAL_DB for full-chat, skipping service registration.")
		return nil
	}

	dbVal, ok := sl.Get(string(registry.DBConnectionKey))
	if !ok {
		return fmt.Errorf("database connection not found in service locator for full-chat module")
	}
	db := dbVal.(*surrealdb.DB)

	slog.Info("Initializing full-chat store")
	store := NewStore(db, fullChatCfg.SurrealNS, fullChatCfg.SurrealDB)
	sl.Set(string(registry.FullChatStoreKey), store)

	return nil
}

// Boot retrieves the service from the container and registers the HTTP routes.
func (m *FullChatModule) Boot(g *echo.Group, sl registry.ServiceLocator) error {
	// Retrieve the store that was created during the Register phase.
	storeVal, ok := sl.Get(string(registry.FullChatStoreKey))
	if !ok || storeVal == nil {
		slog.Info("Full-chat store not available, skipping route registration.")
		return nil
	}
	store := storeVal.(*Store)

	// Create the handler using the store from the service locator.
	handler := NewMessageHandler(store)

	slog.Info("Registering full-chat routes")
	fullChatGroup := g.Group("/full-chat")
	fullChatGroup.GET("", handler.ChatUI)
	fullChatGroup.GET("/ws", handler.WebSocketHandler)

	// API v1 routes
	v1 := fullChatGroup.Group("/api/v1")
	{
		messages := v1.Group("/messages")
		messages.POST("", handler.CreateMessage)
		messages.GET("", handler.ListMessages)
	}
	return nil
}
