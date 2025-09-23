package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/modules/full-chat"
	"github.com/nfrund/goby/internal/modules/wargame"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/templates"
	"github.com/surrealdb/surrealdb.go"
)

// registerModules initializes all application modules that have dependencies
// and returns them for the service locator. This is the central point for
// module registration.
func registerModules(htmlHub, dataHub *hub.Hub, renderer *templates.Renderer, db *surrealdb.DB, cfg config.Provider) map[string]any {
	// Initialize the wargame module
	wargameEngine := wargame.NewEngine(htmlHub, dataHub, renderer)

	// Initialize the full-chat module
	fullChatCfg, err := getFullChatConfig(cfg)
	var fullChatSvc fullchat.Service
	if err != nil {
		slog.Warn("Failed to get full-chat config", "error", err)
	} else if fullChatCfg == nil {
		slog.Warn("Full-chat config is nil")
	} else {
		slog.Info("Initializing full-chat module", 
			"namespace", fullChatCfg.SurrealNS, 
			"database", fullChatCfg.SurrealDB)
		
		// Make sure to use the correct namespace and database for the full-chat module
		if err := db.Use(context.Background(), fullChatCfg.SurrealNS, fullChatCfg.SurrealDB); err != nil {
			slog.Error("Failed to set full-chat database context", "error", err)
		} else {
			slog.Info("Creating full-chat service")
			fullChatSvc = fullchat.NewService(db, fullChatCfg)
		}
	}

	return map[string]any{
		string(registry.WargameEngineKey): wargameEngine,
		string(registry.FullChatService):  fullChatSvc,
	}
}

// getFullChatConfig retrieves and validates the full-chat module configuration.
func getFullChatConfig(cfg config.Provider) (*fullchat.Config, error) {
	// Get the module configuration
	modCfg, exists := cfg.GetModuleConfig("full-chat")
	if !exists {
		return nil, fmt.Errorf("full-chat module configuration not found")
	}

	// Type assert to the expected config type
	fullChatCfg, ok := modCfg.(*fullchat.Config)
	if !ok {
		return nil, fmt.Errorf("invalid full-chat module configuration type")
	}

	// Validate required fields
	if fullChatCfg.SurrealNS == "" || fullChatCfg.SurrealDB == "" {
		return nil, fmt.Errorf("missing required full-chat configuration: SURREAL_NS and SURREAL_DB must be set")
	}

	return fullChatCfg, nil
}
