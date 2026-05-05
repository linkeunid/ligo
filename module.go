package ligo

// Package ligo provides module system for organizing application functionality
// into self-contained units with providers, controllers, and middleware.

import (
	"github.com/linkeunid/ligo/internal/core/lifecycle"
	"github.com/linkeunid/ligo/internal/core/module"
)

// Module represents a self-contained unit of functionality that encapsulates
// providers, controllers, middleware, and lifecycle hooks.
type Module = module.Module

// NewModule creates a new module with the given name and options.
// The name should be unique and descriptive (e.g., "user", "auth", "database").
//
// Example:
//
//	func Module() ligo.Module {
//	    return ligo.NewModule("user",
//	        ligo.Providers(...),
//	        ligo.Controllers(...),
//	    )
//	}
func NewModule(name string, opts ...module.ModuleOption) module.Module {
	return module.New(name, opts...)
}

// Providers adds providers to the module.
// Providers can be Values, Factories, or Transients that are registered
// in the DI container for this module.
//
// Example:
//
//	ligo.Providers(
//	    ligo.Value("config-value"),
//	    ligo.Factory[*UserService](NewUserService),
//	    ligo.Transient[*RequestContext](NewRequestContext),
//	)
func Providers(providers ...any) module.ModuleOption {
	return module.Providers(providers...)
}

// Imports adds child modules to this module.
// Child modules can access exported providers from this module.
//
// Example:
//
//	ligo.Imports(
//	    database.Module(),
//	    auth.Module(),
//	)
func Imports(modules ...module.Module) module.ModuleOption {
	return module.Imports(modules...)
}

// Controllers adds controller constructors that receive dependencies via DI.
// Each constructor is called with resolved dependencies and must return a Controller.
//
// Example:
//
//	ligo.Controllers(
//	    func(svc *UserService) ligo.Controller {
//	        return &UserController{service: svc}
//	    },
//	)
func Controllers(constructors ...any) module.ModuleOption {
	return module.Controllers(constructors...)
}

// Middlewares adds middleware constructors that receive dependencies via DI.
// Each constructor is called with resolved dependencies and must return a Middleware.
//
// Example:
//
//	ligo.Middlewares(
//	    func(logger *Logger) ligo.Middleware {
//	        return LoggingMiddleware(logger)
//	    },
//	)
func Middlewares(constructors ...any) module.ModuleOption {
	return module.Middlewares(constructors...)
}

// OnModuleInit adds a hook to run when the module is initialized.
// Hooks are executed after all providers are registered but before the server starts.
//
// Example:
//
//	ligo.OnModuleInit(func() error {
//	    return database.Connect()
//	})
func OnModuleInit(fn func() error) module.ModuleOption {
	return module.OnModuleInit(fn)
}

// OnModuleDestroy adds a hook to run when the module is destroyed.
// Hooks are executed in reverse order during application shutdown.
//
// Example:
//
//	ligo.OnModuleDestroy(func() error {
//	    return database.Close()
//	})
func OnModuleDestroy(fn func() error) module.ModuleOption {
	return module.OnModuleDestroy(fn)
}

// WithModuleHooks adds explicit lifecycle hooks to the module.
// This is an alternative to OnModuleInit/OnModuleDestroy for more control.
//
// Example:
//
//	ligo.NewModule("user",
//	    ligo.Providers(...),
//	    ligo.WithModuleHooks(
//	        ligo.OnInit(func() error { ... }),
//	        ligo.OnDestroy(func() error { ... }),
//	    ),
//	)
func WithModuleHooks(opts ...ModuleHookOption) module.ModuleOption {
	registry := lifecycle.NewModuleHookRegistry()
	for _, opt := range opts {
		opt(registry)
	}
	return module.Hooks(registry)
}

// Dynamic creates a module option for dynamic modules with configuration options.
// The factory function receives the options and returns a configured module.
// This is useful for creating modules that need runtime configuration.
//
// Example:
//
//	func RegisterConfigModule(folder string) ligo.Module {
//	    return ligo.NewModule("config",
//	        ligo.Dynamic(
//	            NewConfigModule,
//	            folder,
//	        ),
//	    )
//	}
func Dynamic(factory func(...any) module.Module, opts ...any) module.ModuleOption {
	return module.Dynamic(factory, opts...)
}