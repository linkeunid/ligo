package container

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/linkeunid/ligo/internal/core/logger"
)

// Container holds registered providers and resolves dependencies.
type Container struct {
	mu        sync.RWMutex
	parent    *Container
	providers map[reflect.Type]ProviderEntry
	cache     sync.Map // map[reflect.Type]any — thread-safe cache
	locks     sync.Map // map[reflect.Type]*sync.Mutex — per-type lock
	logger    logger.Logger
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
func New(log ...logger.Logger) *Container {
	c := &Container{
		providers: make(map[reflect.Type]ProviderEntry),
	}
	if len(log) > 0 {
		c.logger = log[0]
	}
	return c
}

// NewChild creates a child container that inherits providers from this container.
// Child containers can override parent providers and have their own cache.
func (c *Container) NewChild() *Container {
	child := &Container{
		parent:    c,
		providers: make(map[reflect.Type]ProviderEntry),
	}
	if c.logger != nil {
		child.logger = c.logger
	}
	return child
}

// Types returns all registered types in the container.
func (c *Container) Types() []reflect.Type {
	c.mu.RLock()
	defer c.mu.RUnlock()
	types := make([]reflect.Type, 0, len(c.providers))
	for t := range c.providers {
		types = append(types, t)
	}
	return types
}

// Register adds a provider to the container.
// If a provider for the type already exists, it is ignored and a warning is logged.
func (c *Container) Register(typ reflect.Type, entry ProviderEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.providers[typ]; exists {
		// Duplicate provider - ignore and warn
		if c.logger != nil {
			c.logger.Warn("Duplicate provider ignored, using existing registration", logger.Field{Key: "type", Value: typ.String()})
		}
		return
	}
	c.providers[typ] = entry

	// Log provider registration
	if c.logger != nil {
		name := logger.ExtractProviderName(entry.eager)
		if name == "unknown" && entry.factory != nil {
			name = logger.ExtractProviderName(entry.factory)
		}
		c.logger.Debug("Provider registered", logger.Field{Key: "name", Value: name})
	}
}

// Resolve returns an instance of type T from the container.
func Resolve[T any](c *Container) T {
	typ := reflect.TypeFor[T]()

	instance, err := c.resolve(typ, nil)
	if err != nil {
		panic(err)
	}

	return instance.(T)
}

// ResolveByType returns an instance of the specified type from the container.
// Returns (nil, error) if the type cannot be resolved.
func ResolveByType(c *Container, typ reflect.Type) (any, error) {
	return c.resolve(typ, nil)
}

// resolve resolves a type, tracking the dependency chain for cycle detection.
// Cycle detection uses chain (per-call), NOT global resolving map.
func (c *Container) resolve(typ reflect.Type, chain []reflect.Type) (any, error) {
	for _, t := range chain {
		if t == typ {
			return nil, &ErrCircularDependency{
				Chain: chainToStrings(append(chain, typ)),
			}
		}
	}

	// Fast path: cache hit (handles cached interface aliases too)
	if val, ok := c.cache.Load(typ); ok {
		return val, nil
	}

	c.mu.RLock()
	entry, ok := c.providers[typ]
	c.mu.RUnlock()

	// Interface fallback: scan for a concrete type that implements the interface.
	requestedTyp := typ
	if !ok && typ.Kind() == reflect.Interface {
		var matchType reflect.Type
		var matchEntry ProviderEntry
		var implementors []string

		c.mu.RLock()
		for t, e := range c.providers {
			if t.Implements(typ) {
				implementors = append(implementors, t.String())
				if matchType == nil {
					matchType = t
					matchEntry = e
				}
			}
		}
		c.mu.RUnlock()

		switch len(implementors) {
		case 0:
			// no match — fall through to parent/missing
		case 1:
			entry, ok, typ = matchEntry, true, matchType
			if c.logger != nil {
				c.logger.LogWithContext(logger.ContextDIContainer, "Interface resolved",
					logger.Field{Key: "interface", Value: requestedTyp.String()},
					logger.Field{Key: "concrete", Value: matchType.String()},
				)
			}
		default:
			return nil, &ErrAmbiguousDependency{Interface: typ.String(), Implementors: implementors}
		}
	}

	// If not found in this container, check parent
	if !ok && c.parent != nil {
		return c.parent.resolve(typ, chain)
	}

	if !ok {
		return nil, &ErrMissingDependency{Type: typ.String()}
	}

	// Transient: skip ALL locking and caching — each resolve is independent
	if entry.transient {
		return c.build(typ, entry, chain)
	}

	if val, ok := c.cache.Load(typ); ok {
		return val, nil
	}

	lockIface, _ := c.locks.LoadOrStore(typ, &sync.Mutex{})
	lock := lockIface.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	if val, ok := c.cache.Load(typ); ok {
		return val, nil
	}

	instance, err := c.build(typ, entry, chain)
	if err != nil {
		return nil, err
	}

	c.cache.Store(typ, instance)
	// Also cache under the original interface type so subsequent lookups skip the scan.
	if requestedTyp != typ {
		c.cache.Store(requestedTyp, instance)
	}
	return instance, nil
}

// build constructs an instance from an entry.
func (c *Container) build(typ reflect.Type, entry ProviderEntry, chain []reflect.Type) (any, error) {
	// Eager value (pre-built instance)
	if entry.eager != nil {
		return entry.eager, nil
	}

	// Factory with auto-injection
	if entry.factory != nil {
		if c.logger != nil {
			depNames := make([]string, len(entry.argTypes))
			for i, t := range entry.argTypes {
				depNames[i] = t.String()
			}
			deps := "-"
			if len(depNames) > 0 {
				deps = strings.Join(depNames, ", ")
			}
			c.logger.LogWithContext(logger.ContextDIContainer, "Constructing "+typ.String(),
				logger.Field{Key: "deps", Value: deps},
			)
		}
		newChain := append(chain, typ)
		args := make([]reflect.Value, len(entry.argTypes))
		for i, argType := range entry.argTypes {
			arg, err := c.resolve(argType, newChain)
			if err != nil {
				var missing *ErrMissingDependency
				if errors.As(err, &missing) {
					return nil, &ErrMissingDependency{
						Type:       argType.String(),
						RequiredBy: typ.String(),
						Cause:      err,
					}
				}
				// Only ErrMissingDependency is wrapped to add caller context;
				// other error types already carry full chain info.
				return nil, err
			}
			args[i] = reflect.ValueOf(arg)
		}

		instance, err := entry.factory(args)
		if err != nil {
			return nil, &DIError{
				Type:       typ.String(),
				RequiredBy: "",
				Cause:      err,
			}
		}
		return instance, nil
	}

	return nil, nil
}

// ErrMissingDependency is returned when a required provider is not found.
type ErrMissingDependency struct {
	Type       string
	RequiredBy string
	Cause      error
}

func (e *ErrMissingDependency) Error() string {
	return fmt.Sprintf("ligo: missing dependency %s (required by %s)", e.Type, e.RequiredBy)
}

func (e *ErrMissingDependency) Unwrap() error { return e.Cause }

// ErrCircularDependency is returned when a circular dependency is detected.
type ErrCircularDependency struct {
	Chain []string
}

func (e *ErrCircularDependency) Error() string {
	return fmt.Sprintf("ligo: circular dependency detected: %v", e.Chain)
}

// ErrDuplicateProvider is returned when a provider is registered twice for the same type.
type ErrDuplicateProvider struct {
	Type string
}

func (e *ErrDuplicateProvider) Error() string {
	return fmt.Sprintf("ligo: duplicate provider for type %s", e.Type)
}

// ErrAmbiguousDependency is returned when multiple registered types implement the requested interface.
type ErrAmbiguousDependency struct {
	Interface    string
	Implementors []string
}

func (e *ErrAmbiguousDependency) Error() string {
	return fmt.Sprintf("ligo: ambiguous dependency: multiple types implement %s: %v", e.Interface, e.Implementors)
}

// DIError wraps container resolution failures with context.
type DIError struct {
	Type       string
	RequiredBy string
	Cause      error
}

func (e *DIError) Error() string {
	return fmt.Sprintf("ligo: cannot resolve %s for %s: %v", e.Type, e.RequiredBy, e.Cause)
}

func chainToStrings(chain []reflect.Type) []string {
	strs := make([]string, len(chain))
	for i, t := range chain {
		strs[i] = t.String()
	}
	return strs
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
