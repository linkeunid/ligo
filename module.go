package ligo

import "github.com/linkeunid/ligo/internal/core/module"

// Module represents a self-contained unit of functionality.
type Module = module.Module

// NewModule creates a new module with the given name and options.
func NewModule(name string, opts ...module.ModuleOption) module.Module {
	return module.New(name, opts...)
}

// Providers adds providers to the module.
func Providers(providers ...any) module.ModuleOption {
	return module.Providers(providers...)
}

// Imports adds child modules.
func Imports(modules ...module.Module) module.ModuleOption {
	return module.Imports(modules...)
}

// Controllers adds controller constructors that receive dependencies via DI.
func Controllers(constructors ...any) module.ModuleOption {
	return module.Controllers(constructors...)
}

// Middlewares adds middleware constructors that receive dependencies via DI.
func Middlewares(constructors ...any) module.ModuleOption {
	return module.Middlewares(constructors...)
}

// OnModuleInit adds a hook to run when the module is initialized.
func OnModuleInit(fn func() error) module.ModuleOption {
	return module.OnModuleInit(fn)
}

// OnModuleDestroy adds a hook to run when the module is destroyed.
func OnModuleDestroy(fn func() error) module.ModuleOption {
	return module.OnModuleDestroy(fn)
}

// Dynamic creates a module option for dynamic modules with configuration options.
// The factory function receives the options and returns a configured module.
// Usage: ligo.Dynamic(NewConfigModule, folder)
func Dynamic(factory func(...any) module.Module, opts ...any) module.ModuleOption {
	return module.Dynamic(factory, opts...)
}