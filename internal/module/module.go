package module

import (
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/templates"
)

// Module defines the contract for a self-registering application module.
type Module interface {
	// Name returns a unique identifier for the module.
	Name() string

	// Register is for binding services into the service container.
	// This method should only be used for registration, not for resolving dependencies.
	Register(sl registry.ServiceLocator, cfg config.Provider) error

	// Boot is for using services that have been registered, like setting up routes.
	// This method is called after all modules have been registered.
	Boot(g *echo.Group, sl registry.ServiceLocator) error

	// RegisterTemplates allows a module to register its embedded templates.
	RegisterTemplates(renderer *templates.Renderer)
}
