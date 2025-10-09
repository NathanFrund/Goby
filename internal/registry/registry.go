package registry

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/nfrund/goby/internal/config"
)

// Registry is a type-safe dependency injection container.
type Registry struct {
	mu       sync.RWMutex
	services map[reflect.Type]any
	cfg      config.Provider
}

// New creates a new registry with the application's configuration provider.
func New(cfg config.Provider) *Registry {
	return &Registry{
		services: make(map[reflect.Type]any),
		cfg:      cfg,
	}
}

// Config returns the configuration provider stored in the registry.
func (r *Registry) Config() config.Provider {
	return r.cfg
}

// Set registers a service instance against a specific type.
// ifacePtr should be a nil pointer to the type, e.g. (*MyInterface)(nil) or (**MyConcreteType)(nil)
func (r *Registry) Set(ifacePtr any, impl any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Handle the case where ifacePtr is a nil interface value
	if ifacePtr == nil {
		panic("cannot register service with nil interface pointer")
	}

	// Get the type of the pointer's element
	ptrType := reflect.TypeOf(ifacePtr)
	if ptrType.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("expected pointer type, got %T", ifacePtr))
	}

	elemType := ptrType.Elem()
	r.services[elemType] = impl
}

// Get retrieves a service from the registry by its type.
// T can be either an interface type or a concrete type.
func Get[T any](r *Registry) (T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var iface T
	// Get the type of T
	typ := reflect.TypeOf((*T)(nil)).Elem()

	service, exists := r.services[typ]
	if !exists {
		return iface, fmt.Errorf("service not found: %v", typ)
	}

	// Convert the service to the expected type
	result, ok := service.(T)
	if !ok {
		return iface, fmt.Errorf("type assertion failed: %T is not %T", service, iface)
	}

	return result, nil
}

// MustGet retrieves a service or panics if not found. This is useful for
// wiring up essential dependencies at startup.
func MustGet[T any](r *Registry) T {
	service, err := Get[T](r)
	if err != nil {
		panic(fmt.Sprintf("failed to get service: %v", err))
	}
	return service
}
