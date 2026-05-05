package module

import "github.com/linkeunid/ligo/internal/core/lifecycle"

// DynamicModule wraps a module factory with options.
type DynamicModule struct {
	Factory func(...any) Module
	Options []any
}

// Module represents a self-contained unit of functionality.
type Module struct {
	Name        string
	Providers   []any // Provider
	Controllers []ControllerConstructor
	Imports     []Module
	Middlewares []MiddlewareConstructor
	OnInit      []func() error
	OnDestroy   []func() error
	Hooks       *lifecycle.ModuleHookRegistry
	Dynamic     *DynamicModule
}

// MiddlewareConstructor holds a middleware constructor.
type MiddlewareConstructor struct {
	Fn any
}

// ControllerConstructor holds a controller constructor.
type ControllerConstructor struct {
	Fn any
}

// ModuleOption configures a Module.
type ModuleOption func(*Module)

// New creates a new module with the given name and options.
func New(name string, opts ...ModuleOption) Module {
	m := Module{Name: name}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// Providers adds providers to the module.
func Providers(providers ...any) ModuleOption {
	return func(m *Module) {
		m.Providers = append(m.Providers, providers...)
	}
}

// Imports adds child modules.
func Imports(modules ...Module) ModuleOption {
	return func(m *Module) {
		m.Imports = append(m.Imports, modules...)
	}
}

// Controllers adds controller constructors that receive dependencies via DI.
func Controllers(constructors ...any) ModuleOption {
	return func(m *Module) {
		for _, c := range constructors {
			m.Controllers = append(m.Controllers, ControllerConstructor{Fn: c})
		}
	}
}

// Middlewares adds middleware constructors that receive dependencies via DI.
func Middlewares(constructors ...any) ModuleOption {
	return func(m *Module) {
		for _, c := range constructors {
			m.Middlewares = append(m.Middlewares, MiddlewareConstructor{Fn: c})
		}
	}
}

// OnModuleInit adds a hook to run when the module is initialized.
func OnModuleInit(fn func() error) ModuleOption {
	return func(m *Module) {
		m.OnInit = append(m.OnInit, fn)
	}
}

// OnModuleDestroy adds a hook to run when the module is destroyed.
func OnModuleDestroy(fn func() error) ModuleOption {
	return func(m *Module) {
		m.OnDestroy = append(m.OnDestroy, fn)
	}
}

// Hooks adds explicit lifecycle hooks to the module.
func Hooks(registry *lifecycle.ModuleHookRegistry) ModuleOption {
	return func(m *Module) {
		m.Hooks = registry
	}
}

// Dynamic creates a module option for dynamic modules.
// The factory function receives the options and returns a configured module.
// Usage: ligo.Dynamic(NewConfigModule, folder)
func Dynamic(factory func(...any) Module, opts ...any) ModuleOption {
	return func(m *Module) {
		m.Dynamic = &DynamicModule{
			Factory: factory,
			Options: opts,
		}
	}
}
