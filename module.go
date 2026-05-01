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