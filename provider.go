package ligo

// Package ligo provides dependency injection providers for registering
// values, factories, and transient services in the DI di.

import (
	"reflect"

	"github.com/linkeunid/ligo/internal/core/lifecycle"
)

// Provider represents a dependency provider that can be registered
// in the DI di. Providers can be eager values or factory functions.
type Provider struct {
	typ          reflect.Type
	fn           any // raw factory function
	eager        any
	transient    bool
	exported     bool
	eagerResolve bool
	hooks        *lifecycle.HookRegistry
}

// Value registers a pre-built instance as a singleton.
// The same instance will be returned for all resolutions of this type.
//
// Example:
//
//	ligo.Value("config-value")
//	ligo.Value(&Config{Debug: true})
//
// With hooks:
//
//	ligo.Value(&Database{db: db}, ligo.WithHooks(
//	    ligo.OnShutdown(func(db *Database) error { return db.Close() }),
//	))
func Value[T any](instance T, opts ...ProviderOption) Provider {
	p := Provider{
		typ:   reflect.TypeFor[T](),
		eager: instance,
	}
	for _, opt := range opts {
		opt(&p)
	}
	return p
}

// Factory registers a factory function that produces a singleton.
// The function can have dependencies as parameters; they are auto-injected.
// The factory is called once, and the result is cached for subsequent resolutions.
//
// Hooks can be attached using WithHooks:
//
//	ligo.Factory[*UserService](func(repo *UserRepository) *UserService {
//	    return NewUserService(repo)
//	}, ligo.WithHooks(
//	    ligo.OnInit(func(svc *UserService) error { ... }),
//	))
func Factory[T any](fn any, opts ...ProviderOption) Provider {
	p := Provider{
		typ: reflect.TypeFor[T](),
		fn:  fn,
	}
	for _, opt := range opts {
		opt(&p)
	}
	return p
}

// Transient registers a factory function that produces a new instance on each resolve.
// Unlike Factory, the factory function is called every time the type is resolved.
// Dependencies are still auto-injected.
//
// Example:
//
//	ligo.Transient[*RequestContext](func() *RequestContext {
//	    return NewRequestContext()
//	}, ligo.WithHooks(
//	    ligo.OnInit(func(ctx *RequestContext) error { ... }),
//	))
func Transient[T any](fn any, opts ...ProviderOption) Provider {
	p := Factory[T](fn, opts...)
	p.transient = true
	return p
}

// HookedFactory registers a factory function and enables explicit hook registration
// via a Register method on the created instance. This provides compile-time safety
// for hook method expressions.
//
// The created instance can implement:
//
//	type Registerable interface {
//	    Register(*lifecycle.HookRegistry)
//	}
//
// Example:
//
//	type Database struct {
//	    db *sql.DB
//	}
//
//	func (d *Database) Connect() error {
//	    d.db = sql.Open("postgres", "dsn")
//	    return nil
//	}
//
//	func (d *Database) Close() error {
//	    return d.db.Close()
//	}
//
//	// Register method enables compile-time safe hook registration
//	func (d *Database) Register(r *lifecycle.HookRegistry) {
//	    r.OnInit(d.Connect)    // Method expression - compile-time checked
//	    r.OnShutdown(d.Close)  // If Close doesn't exist → compile error
//	}
//
//	// Provider registration
//	ligo.HookedFactory(NewDatabase)
func HookedFactory[T any](fn any) Provider {
	return Factory[T](fn, WithHooks())
}

// HookedSingleton is like [HookedFactory] but marks the provider for eager
// resolution at application startup. Use it for providers whose only purpose
// is to attach lifecycle hooks (RPC handler registrations, background
// workers, schedulers) — where nothing else in the DI graph depends on the
// type, so a plain [HookedFactory] would never be instantiated and its
// Register method would never fire.
//
// Example:
//
//	// OrderMessaging only exists to bind RPC handlers in OnBootstrap. No
//	// other provider depends on it, so HookedFactory would be a no-op.
//	ligo.HookedSingleton[*OrderMessaging](NewOrderMessaging)
//
// Eager providers are resolved after all modules have been built and before
// any OnInit / OnBootstrap hook executes, so their dependencies (and the
// hooks they register) participate normally in the lifecycle.
func HookedSingleton[T any](fn any) Provider {
	p := Factory[T](fn, WithHooks())
	p.eagerResolve = true
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

// IsEagerResolve returns true if the provider should be resolved at startup
// even when nothing else in the DI graph depends on it. Set by [HookedSingleton].
func (p Provider) IsEagerResolve() bool {
	return p.eagerResolve
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

// Hooks returns the hook registry if set (for internal use).
func (p Provider) Hooks() *lifecycle.HookRegistry {
	return p.hooks
}

// ProviderOption is a functional option for configuring providers.
type ProviderOption func(*Provider)

// WithHooks attaches lifecycle hooks to a provider.
//
// Example:
//
//	ligo.Factory[*Database](NewDatabase,
//	    ligo.WithHooks(
//	        ligo.OnInit(func(db *Database) error {
//	            var err error
//	            db.conn = sql.Open("postgres", "dsn")
//	            return err
//	        }),
//	        ligo.BeforeShutdown(func(db *Database) error {
//	            return db.conn.Close()
//	        }),
//	    ),
//	)
func WithHooks(hooks ...HookOption) ProviderOption {
	registry := Hooks(hooks...)
	return func(p *Provider) {
		p.hooks = registry
	}
}
