package script

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/registry"
)

// KeyEngine is the type-safe key for accessing the script engine service from the registry.
var KeyEngine = registry.Key[ScriptEngine]("core.script.Engine")

// RegisterService registers the script engine in the application registry
func RegisterService(reg *registry.Registry, cfg config.Provider) (*Engine, error) {
	slog.Info("Registering script engine service")

	// Create the script engine
	engine := NewEngine(Dependencies{
		Config: cfg,
	})

	// Check hot-reload configuration
	hotReloadEnabled := os.Getenv("HOT_RELOAD_SCRIPTS") != "false" // Default to true
	slog.Info("Script engine configuration", "hot_reload_enabled", hotReloadEnabled)

	// Initialize the engine
	if err := engine.Initialize(context.Background(), hotReloadEnabled); err != nil {
		return nil, err
	}

	// Register the engine in the registry
	registry.Set[ScriptEngine](reg, KeyEngine, engine)

	slog.Info("Script engine service registered successfully")
	return engine, nil
}

// GetService retrieves the script engine from the registry
func GetService(reg *registry.Registry) (ScriptEngine, error) {
	engine, ok := registry.Get(reg, KeyEngine)
	if !ok {
		return nil, fmt.Errorf("script engine not found in registry")
	}
	return engine, nil
}

// MustGetService retrieves the script engine from the registry or panics
func MustGetService(reg *registry.Registry) ScriptEngine {
	return registry.MustGet(reg, KeyEngine)
}
