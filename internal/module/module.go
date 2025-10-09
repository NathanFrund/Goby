package module

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/registry"
)

// Module defines the contract for a self-contained application feature.
type Module interface {
	// Name returns a unique identifier for the module.
	Name() string

	// Register is called during application startup to register the module's
	// services with the central registry.
	Register(reg *registry.Registry) error

	// Boot is called after all modules have registered their services.
	// This is the phase for setting up routes and starting background processes.
	Boot(ctx context.Context, router *echo.Group, reg *registry.Registry) error

	// Shutdown is called during graceful application shutdown.
	// This is the phase for cleaning up resources and stopping background processes.
	Shutdown(ctx context.Context) error
}

// BaseModule provides default no-op implementations for Module methods.
// Modules can embed this to avoid implementing methods they don't need.
type BaseModule struct{}

func (m *BaseModule) Register(reg *registry.Registry) error { return nil }
func (m *BaseModule) Boot(ctx context.Context, router *echo.Group, reg *registry.Registry) error {
	return nil
}
func (m *BaseModule) Shutdown(ctx context.Context) error {
	return nil
}
