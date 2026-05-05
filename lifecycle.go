package ligo

import "github.com/linkeunid/ligo/internal/core/lifecycle"

// Lifecycle hooks allow providers and controllers to execute code at specific
// application lifecycle stages.
//
// There are three ways to use lifecycle hooks:
//
// 1. **Interface-based (implicit)** - Implement hook methods directly on your
//    provider or controller structs. The framework will automatically detect
//    and execute them at the appropriate time.
//
// 2. **Explicit registration** - Use Hooks() to explicitly register hook functions.
//
// 3. **Compile-time safe registration** - Implement the Registerable interface
//    and use HookedFactory/Value(WithHooks()) for compile-time checked hook methods.
//
// ## Interface-based Example:
//
//	type DatabaseService struct {
//	    db *sql.DB
//	}
//
//	func (s *DatabaseService) OnModuleInit() error {
//	    var err error
//	    s.db = sql.Open("postgres", "dsn")
//	    return err
//	}
//
//	func (s *DatabaseService) BeforeApplicationShutdown() error {
//	    // Stop accepting new connections, finish in-flight requests
//	    return s.db.Close()
//	}
//
//	func (s *DatabaseService) OnApplicationShutdown() error {
//	    // Final cleanup after all connections are drained
//	    return nil
//	}
//
// ## Explicit Registration Example:
//
//	ligo.Factory[*DatabaseService](NewDatabaseService,
//	   	ligo.Hooks(
//	   		ligo.OnInit(func(db *DatabaseService) error {
//	   			var err error
//	   			db.db = sql.Open("postgres", "dsn")
//	   			return err
//	   		}),
//	   		ligo.BeforeShutdown(func(db *DatabaseService) error {
//	   			return db.db.Close()
//	   		}),
//	   	),
//	)
//
// ## Compile-Time Safe Registration (HookedFactory):
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
//	// Register implements the Registerable interface for compile-time safe hook registration.
//	func (d *Database) Register(r *ligo.HookRegistry) {
//	    r.OnInit(d.Connect)     // If Connect doesn't exist → compile error
//	    r.OnShutdown(d.Close)   // Typo "Conenct" → compile error
//	}
//
//	// Provider registration with HookedFactory
//	ligo.HookedFactory[*Database](NewDatabase)
//	// OR with Value:
//	ligo.Value(database, ligo.WithHooks())
//
// ## Module-level Hooks Example:
//
//	ligo.NewModule("user",
//	   	ligo.Providers(...),
//	   	ligo.ModuleHooks(
//	   		ligo.OnInit(func() error {
//	   			// Run migrations
//	   			return Migrate()
//	   		}),
//	   		ligo.OnDestroy(func() error {
//	   			// Cleanup
//	   			return Cleanup()
//	   		}),
//	   	),
//	)
//
// Available hooks (in execution order):
//
//   - OnModuleInit() error — Called when the module containing this provider
//     is initialized. Runs per-module, depth-first during app startup.
//
//   - OnApplicationBootstrap() error — Called after all modules are initialized,
//     but before the application starts serving (HTTP or signals).
//
//   - BeforeApplicationShutdown() error — Called before shutdown begins,
//     before OnApplicationShutdown. Useful for graceful drain-stop scenarios.
//     Runs once in reverse order.
//
//   - OnApplicationShutdown() error — Called during application shutdown,
//     after BeforeApplicationShutdown, before OnModuleDestroy. Runs once in reverse order.
//
//   - OnModuleDestroy() error — Called when the module containing this provider
//     is destroyed. Runs per-module, reverse depth-first during shutdown.

// HookFunc represents a lifecycle hook function.
type HookFunc = lifecycle.HookFunc

// HookRegistry stores lifecycle hooks for a specific type.
// Use Hooks() to create a new registry.
type HookRegistry = lifecycle.HookRegistry

// HookRegistryRef is a pointer to HookRegistry for use in Register methods.
// This is a convenience type alias for method receivers.
type HookRegistryRef = *lifecycle.HookRegistry

// Hooks creates a new hook registry for explicit lifecycle hook registration.
// Use with providers and controllers:
//
//	ligo.Factory[*Database](NewDatabase,
//	    ligo.Hooks(
//	        ligo.OnInit(func(db *Database) error { ... }),
//	        ligo.BeforeShutdown(func(db *Database) error { ... }),
//	    ),
//	)
func Hooks(hooks ...HookOption) *lifecycle.HookRegistry {
	registry := lifecycle.NewHookRegistry(nil)
	for _, h := range hooks {
		h(registry)
	}
	return registry
}

// HookOption is a functional option for configuring hooks.
type HookOption func(*lifecycle.HookRegistry)

// OnInit creates a hook option for OnModuleInit.
func OnInit(fn func() error) HookOption {
	return func(r *lifecycle.HookRegistry) {
		r.OnInit(fn)
	}
}

// OnBootstrap creates a hook option for OnApplicationBootstrap.
func OnBootstrap(fn func() error) HookOption {
	return func(r *lifecycle.HookRegistry) {
		r.OnBootstrap(fn)
	}
}

// BeforeShutdown creates a hook option for BeforeApplicationShutdown.
func BeforeShutdown(fn func() error) HookOption {
	return func(r *lifecycle.HookRegistry) {
		r.BeforeShutdown(fn)
	}
}

// OnShutdown creates a hook option for OnApplicationShutdown.
func OnShutdown(fn func() error) HookOption {
	return func(r *lifecycle.HookRegistry) {
		r.OnShutdown(fn)
	}
}

// OnDestroy creates a hook option for OnModuleDestroy.
func OnDestroy(fn func() error) HookOption {
	return func(r *lifecycle.HookRegistry) {
		r.OnDestroy(fn)
	}
}

// ModuleHookOption is a functional option for module-level hooks.
type ModuleHookOption func(*lifecycle.ModuleHookRegistry)

// ModuleInit creates a module-level hook option for OnModuleInit.
func ModuleInit(fn func() error) ModuleHookOption {
	return func(r *lifecycle.ModuleHookRegistry) {
		r.OnInit(fn)
	}
}

// ModuleDestroy creates a module-level hook option for OnModuleDestroy.
func ModuleDestroy(fn func() error) ModuleHookOption {
	return func(r *lifecycle.ModuleHookRegistry) {
		r.OnDestroy(fn)
	}
}

// ModuleHooks creates module-level lifecycle hooks.
// Use with NewModule:
//
//	ligo.NewModule("user",
//	    ligo.Providers(...),
//	    ligo.ModuleHooks(
//	        ligo.ModuleInit(func() error { ... }),
//	        ligo.ModuleDestroy(func() error { ... }),
//	    ),
//	)
func ModuleHooks(opts ...ModuleHookOption) *lifecycle.ModuleHookRegistry {
	registry := lifecycle.NewModuleHookRegistry()
	for _, opt := range opts {
		opt(registry)
	}
	return registry
}

// CollectHooks is the internal hook collection function.
// Re-exported for internal use.
func CollectHooks(v any) lifecycle.Hooks {
	return lifecycle.CollectHooks(v)
}

// Registerable is the interface that services can implement to explicitly
// register their lifecycle hooks. This enables compile-time safety via method
// expressions instead of relying on duck typing.
//
// Example:
//
//	type Database struct {}
//
//	func (d *Database) Connect() error { ... }
//	func (d *Database) Close() error { ... }
//
//	func (d *Database) Register(r *lifecycle.HookRegistry) {
//	    r.OnInit(d.Connect)      // Method expression - compile-time checked
//	    r.OnShutdown(d.Close)    // If Close doesn't exist → compile error
//	}
type Registerable = lifecycle.Registerable
