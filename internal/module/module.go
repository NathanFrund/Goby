package module

import (
	"io/fs"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/registry"
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

	// TemplateFS returns the filesystem containing the module's templates.
	// Return nil if the module doesn't provide any templates.
	TemplateFS() fs.FS
}
