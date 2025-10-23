package script

import (
	"context"
	"log/slog"

	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/registry"
)

// RegisterService registers the script engine in the application registry
func RegisterService(reg *registry.Registry, cfg config.Provider) (*Engine, error) {
	slog.Info("Registering script engine service")

	// Create the script engine
	engine := NewEngine(Dependencies{
		Config: cfg,
	})

	// Initialize the engine
	if err := engine.Initialize(context.Background()); err != nil {
		return nil, err
	}

	// Register the engine in the registry
	reg.Set((*ScriptEngine)(nil), engine)

	slog.Info("Script engine service registered successfully")
	return engine, nil
}

// GetService retrieves the script engine from the registry
func GetService(reg *registry.Registry) (ScriptEngine, error) {
	return registry.Get[ScriptEngine](reg)
}

// MustGetService retrieves the script engine from the registry or panics
func MustGetService(reg *registry.Registry) ScriptEngine {
	return registry.MustGet[ScriptEngine](reg)
}