package registry

import (
	"log/slog"

	"github.com/labstack/echo/v4"
)

// ServiceLocator defines a simple interface for dependency injection into modules.
// It allows modules to retrieve shared services by name without a direct import.
type ServiceLocator interface {
	Get(name string) any
}

// RouteRegistrar is a function that registers a module's routes.
type RouteRegistrar func(group *echo.Group, sl ServiceLocator)

var registrars []RouteRegistrar

// Register adds a route registrar function to the global list.
// This is intended to be called from the init() function of a module's routes package.
func Register(rr RouteRegistrar) {
	registrars = append(registrars, rr)
}

// Apply iterates over all registered RouteRegistrars and executes them,
// wiring the module routes into the provided Echo group.
func Apply(group *echo.Group, sl ServiceLocator) {
	for _, rr := range registrars {
		rr(group, sl)
	}
	slog.Info("Applied all registered module routes", "count", len(registrars))
}
