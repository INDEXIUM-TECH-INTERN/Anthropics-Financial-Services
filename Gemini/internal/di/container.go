package di

import (
	"reflect"
	"sync"
)

// Container represents the DI container
type Container struct {
	services map[reflect.Type]interface{}
	mu       sync.RWMutex
}

// NewContainer creates a new DI container
func NewContainer() *Container {
	return &Container{
		services: make(map[reflect.Type]interface{}),
	}
}

// Register registers a service with the container
func (c *Container) Register(service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	typ := reflect.TypeOf(service)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	c.services[typ] = service
}

// RegisterNamed registers a service with a name
func (c *Container) RegisterNamed(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	typ := reflect.TypeOf(service)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	key := reflect.TypeOf((*namedService)(nil)).Elem()
	c.services[key] = &namedService{
		name:    name,
		service: service,
		typ:     typ,
	}
}

// Resolve resolves a service by type
func (c *Container) Resolve(typ reflect.Type) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if it's a named service type
	if typ == reflect.TypeOf((*namedService)(nil)).Elem() {
		if service, ok := c.services[typ]; ok {
			return service, true
		}
	}

	// Try to resolve by type
	if service, ok := c.services[typ]; ok {
		return service, true
	}

	// Try to resolve by pointer type
	if typ.Kind() == reflect.Interface {
		for key, service := range c.services {
			if key.Implements(typ) {
				return service, true
			}
		}
	}

	return nil, false
}

// ResolveByName resolves a service by name
func (c *Container) ResolveByName(name string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := reflect.TypeOf((*namedService)(nil)).Elem()
	if service, ok := c.services[key]; ok {
		if ns, ok := service.(*namedService); ok && ns.name == name {
			return ns.service, true
		}
	}

	return nil, false
}

// ResolveAll resolves all services that implement a type
func (c *Container) ResolveAll(typ reflect.Type) []interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var services []interface{}
	for key, service := range c.services {
		if typ.Kind() == reflect.Interface && key.Implements(typ) {
			services = append(services, service)
		}
	}

	return services
}

// Has checks if a service is registered
func (c *Container) Has(typ reflect.Type) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.services[typ]
	return exists
}

// Clear removes all services
func (c *Container) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services = make(map[reflect.Type]interface{})
}

// namedService represents a named service
type namedService struct {
	name    string
	service interface{}
	typ     reflect.Type
}

// IsNamedService checks if a type is a named service
func IsNamedService(typ reflect.Type) bool {
	return typ == reflect.TypeOf((*namedService)(nil)).Elem()
}