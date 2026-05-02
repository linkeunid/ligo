package ligo

// Package ligo provides dependency injection providers for registering
// values, factories, and transient services in the DI container.

import "reflect"

// Provider represents a dependency provider that can be registered
// in the DI container. Providers can be eager values or factory functions.
type Provider struct {
	typ       reflect.Type
	fn        any // raw factory function
	eager     any
	transient bool
	exported  bool
}

// Value registers a pre-built instance as a singleton.
// The same instance will be returned for all resolutions of this type.
//
// Example:
//
//	ligo.Value("config-value")
//	ligo.Value(&Config{Debug: true})
func Value[T any](instance T) Provider {
	return Provider{
		typ:   reflect.TypeFor[T](),
		eager: instance,
	}
}

// Factory registers a factory function that produces a singleton.
// The function can have dependencies as parameters; they are auto-injected.
// The factory is called once, and the result is cached for subsequent resolutions.
//
// Example:
//
//	ligo.Factory[*UserService](func(repo *UserRepository) *UserService {
//	    return NewUserService(repo)
//	})
func Factory[T any](fn any) Provider {
	return Provider{
		typ: reflect.TypeFor[T](),
		fn:  fn,
	}
}

// Transient registers a factory function that produces a new instance on each resolve.
// Unlike Factory, the factory function is called every time the type is resolved.
// Dependencies are still auto-injected.
//
// Example:
//
//	ligo.Transient[*RequestContext](func() *RequestContext {
//	    return NewRequestContext()
//	})
func Transient[T any](fn any) Provider {
	p := Factory[T](fn)
	p.transient = true
	return p
}

// Type returns the type this provider produces.
func (p Provider) Type() reflect.Type {
	return p.typ
}

// IsExported returns true if the provider is exported to sibling modules.
// Exported providers are visible to modules that import the module that exports them.
func (p Provider) IsExported() bool {
	return p.exported
}

// IsTransient returns true if the provider creates new instances per resolve.
func (p Provider) IsTransient() bool {
	return p.transient
}

// Export marks a provider as exported, making it visible to sibling modules.
// This allows providers to be shared across modules without being global.
//
// Example:
//
//	ligo.Export(ligo.Factory[*Database](func() *Database {
//	    return NewDatabase()
//	}))
func Export(p Provider) Provider {
	p.exported = true
	return p
}

// Fn returns the factory function (for internal use).
func (p Provider) Fn() any {
	return p.fn
}

// Eager returns the eager value if set (for internal use).
func (p Provider) Eager() any {
	return p.eager
}
