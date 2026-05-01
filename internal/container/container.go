package container

import (
	"fmt"
	"reflect"
)

// Container holds registered providers and resolves dependencies.
type Container struct {
	providers map[reflect.Type]ProviderEntry
	cache     map[reflect.Type]any
}

// ProviderEntry represents a registered provider in the container.
type ProviderEntry struct {
	factory   func(args []reflect.Value) (any, error)
	eager     any
	argTypes  []reflect.Type
	transient bool
	exported  bool
}

// New creates a new DI container.
func New() *Container {
	return &Container{
		providers: make(map[reflect.Type]ProviderEntry),
		cache:     make(map[reflect.Type]any),
	}
}

// Register adds a provider to the container.
func (c *Container) Register(typ reflect.Type, entry ProviderEntry) {
	if _, exists := c.providers[typ]; exists {
		panic(fmt.Errorf("ligo: duplicate provider for type %s", typ.String()))
	}
	c.providers[typ] = entry
}

// Resolve returns an instance of type T from the container.
func Resolve[T any](c *Container) T {
	var zero T
	typ := reflect.TypeOf(zero)

	instance := c.resolve(typ, nil)
	if instance == nil {
		panic(fmt.Errorf("ligo: missing dependency %s", typ.String()))
	}

	return instance.(T)
}

// ResolveByType returns an instance of the specified type from the container.
// Returns nil if the type is not registered.
func ResolveByType(c *Container, typ reflect.Type) any {
	return c.resolve(typ, nil)
}

// resolve resolves a type, tracking the dependency chain for cycle detection.
func (c *Container) resolve(typ reflect.Type, chain []reflect.Type) any {
	// Check cache first
	if instance, ok := c.cache[typ]; ok {
		return instance
	}

	// Check local providers
	entry, ok := c.providers[typ]
	if !ok {
		return nil
	}

	// Detect cycle
	for _, t := range chain {
		if t == typ {
			chainStrs := make([]string, len(chain)+1)
			for i, ct := range chain {
				chainStrs[i] = ct.String()
			}
			chainStrs[len(chain)] = typ.String()
			panic(fmt.Errorf("ligo: circular dependency detected: %v", chainStrs))
		}
	}

	// Eager value
	if entry.eager != nil {
		c.cache[typ] = entry.eager
		return entry.eager
	}

	// Factory with auto-injection
	if entry.factory != nil {
		newChain := append(chain, typ)
		args := make([]reflect.Value, len(entry.argTypes))
		for i, argType := range entry.argTypes {
			arg := c.resolve(argType, newChain)
			if arg == nil {
				panic(fmt.Errorf("ligo: missing dependency %s (required by %s)", argType.String(), typ.String()))
			}
			args[i] = reflect.ValueOf(arg)
		}

		instance, err := entry.factory(args)
		if err != nil {
			panic(fmt.Errorf("ligo: failed to create %s: %w", typ, err))
		}
		if !entry.transient {
			c.cache[typ] = instance
		}
		return instance
	}

	return nil
}

// NewEntry creates a provider entry for registration.
func NewEntry(factory func(args []reflect.Value) (any, error), eager any, argTypes []reflect.Type, transient, exported bool) ProviderEntry {
	return ProviderEntry{
		factory:   factory,
		eager:     eager,
		argTypes:  argTypes,
		transient: transient,
		exported:  exported,
	}
}