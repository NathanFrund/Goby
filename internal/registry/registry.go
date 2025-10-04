package registry

import (
	"sync"
)

// ServiceLocator defines an interface for a service locator.
type ServiceLocator interface {
	Get(key string) (any, bool)
	Set(key string, service any)
}

// --- New Service Locator Implementation ---

// serviceLocator is a thread-safe implementation of the ServiceLocator interface.
type serviceLocator struct {
	mu       sync.RWMutex
	services map[string]any
}

// NewServiceLocator creates a new instance of a service locator.
func NewServiceLocator() ServiceLocator {
	return &serviceLocator{
		services: make(map[string]any),
	}
}

// Get retrieves a service by its key.
func (sl *serviceLocator) Get(key string) (any, bool) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	service, ok := sl.services[key]
	return service, ok
}

// Set registers a service with a given key.
func (sl *serviceLocator) Set(key string, service any) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.services[key] = service
}
