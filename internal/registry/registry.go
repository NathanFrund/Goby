package registry

import (
	"fmt"
	"sync"

	"github.com/nfrund/goby/internal/config"
)

// Key is a type-safe, generic key for registering and retrieving services.
// The string value should be a unique identifier, e.g., "moduleName.serviceName".
type Key[T any] string

// Registry provides a type-safe way for modules to share and discover services at runtime.
// It uses a sync.Map for concurrent-safe access.
type Registry struct {
	services sync.Map
	cfg      config.Provider
}

// New creates a new registry with the application's configuration provider.
func New(cfg config.Provider) *Registry {
	return &Registry{
		cfg: cfg,
	}
}

// Config returns the configuration provider stored in the registry.
func (r *Registry) Config() config.Provider {
	return r.cfg
}

// Set registers a service instance against a type-safe key.
func Set[T any](r *Registry, key Key[T], value T) {
	r.services.Store(key, value)
}

// Get retrieves a service from the registry by its type.
func Get[T any](r *Registry, key Key[T]) (T, bool) {
	val, ok := r.services.Load(string(key))
	if !ok {
		var zero T
		return zero, false
	}

	// Convert the service to the expected type
	result, ok := val.(T)
	if !ok {
		// This should ideally never happen if keys are used correctly,
		// but it's a good safeguard.
		var zero T
		return zero, false
	}

	return result, true
}

// MustGet retrieves a service or panics if not found. This is useful for
// wiring up essential dependencies at startup.
func MustGet[T any](r *Registry, key Key[T]) T {
	val, ok := Get(r, key)
	if !ok {
		panic(fmt.Sprintf("service not found for key: %v", key))
	}
	return val
}
