package module

import "github.com/linkeunid/ligo/internal/core/lifecycle"

// DynamicModule wraps a module factory with options.
type DynamicModule struct {
	Factory func(...any) Module
	Options []any
}

// Module represents a self-contained unit of functionality.
//
// All module-level lifecycle hooks (whether added via OnModuleInit /
// OnModuleDestroy options or via an explicit ligo.Hooks(registry) option)
// land on Hooks. There is no longer a parallel OnInit/OnDestroy slice on
// the Module struct itself.
type Module struct {
	Name        string
	Providers   []any // Provider
	Controllers []ControllerConstructor
	Imports     []Module
	Middlewares []MiddlewareConstructor
	Hooks       *lifecycle.ModuleHookRegistry
	Dynamic     *DynamicModule
}

// ensureHooks lazily allocates the hook registry on first use so that
// callers do not need to wire one up explicitly.
func (m *Module) ensureHooks() *lifecycle.ModuleHookRegistry {
	if m.Hooks == nil {
		m.Hooks = lifecycle.NewModuleHookRegistry()
	}
	return m.Hooks
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
// Multiple OnModuleInit options on the same module all run in registration order.
func OnModuleInit(fn func() error) ModuleOption {
	return func(m *Module) {
		m.ensureHooks().OnInit(fn)
	}
}

// OnModuleDestroy adds a hook to run when the module is destroyed.
// Multiple OnModuleDestroy options on the same module all run in registration order.
func OnModuleDestroy(fn func() error) ModuleOption {
	return func(m *Module) {
		m.ensureHooks().OnDestroy(fn)
	}
}

// Hooks installs an explicit lifecycle hook registry on the module. Hooks
// already attached via OnModuleInit / OnModuleDestroy are merged into the
// supplied registry rather than discarded.
func Hooks(registry *lifecycle.ModuleHookRegistry) ModuleOption {
	return func(m *Module) {
		if registry == nil {
			return
		}
		if m.Hooks != nil {
			for _, fn := range m.Hooks.GetInitHooks() {
				registry.OnInit(fn)
			}
			for _, fn := range m.Hooks.GetDestroyHooks() {
				registry.OnDestroy(fn)
			}
		}
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
