package ligo

import (
	"fmt"
	"reflect"

	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/module"
)

// registerProvider registers a provider in the container.
func (a *App) registerProvider(c *container.Container, p Provider) {
	entry := a.buildProviderEntry(p)
	c.Register(p.Type(), entry)
}

// buildProviderEntry builds a container entry from a provider.
func (a *App) buildProviderEntry(p Provider) container.ProviderEntry {
	if p.Eager() != nil {
		return container.NewEntry(nil, p.Eager(), nil, p.IsTransient(), p.IsExported())
	}

	// Factory with auto-injection
	fnValue := reflect.ValueOf(p.fn)
	fnType := fnValue.Type()

	argTypes := make([]reflect.Type, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		argTypes[i] = fnType.In(i)
	}

	return container.NewEntry(func(args []reflect.Value) (any, error) {
		out := fnValue.Call(args)
		if len(out) == 0 {
			return nil, fmt.Errorf("ligo: factory function must return a value")
		}
		return out[0].Interface(), nil
	}, nil, argTypes, p.transient, p.exported)
}

// buildModule registers providers from a module and its imports in the container.
func (a *App) buildModule(parent *container.Container, mod module.Module) {
	// Handle dynamic modules
	if mod.Dynamic != nil {
		dynamicMod := mod.Dynamic.Factory(mod.Dynamic.Options...)
		// Merge dynamic module with current module
		mod.Providers = append(mod.Providers, dynamicMod.Providers...)
		mod.Controllers = append(mod.Controllers, dynamicMod.Controllers...)
		mod.Imports = append(mod.Imports, dynamicMod.Imports...)
		mod.Middlewares = append(mod.Middlewares, dynamicMod.Middlewares...)
		mod.OnInit = append(mod.OnInit, dynamicMod.OnInit...)
		mod.OnDestroy = append(mod.OnDestroy, dynamicMod.OnDestroy...)
		// Note: Dynamic modules don't merge further dynamic modules
	}

	modContainer := parent // flat graph - modules share root container

	// Register module providers
	for _, p := range mod.Providers {
		provider := p.(Provider)
		if provider.IsExported() {
			a.registerProvider(parent, provider)
		} else {
			a.registerProvider(modContainer, provider)
		}
	}

	// Collect module lifecycle hooks
	if len(mod.OnInit) > 0 {
		a.moduleHooks.onInit = append(a.moduleHooks.onInit, mod.OnInit)
	}
	if len(mod.OnDestroy) > 0 {
		a.moduleHooks.onDestroy = append(a.moduleHooks.onDestroy, mod.OnDestroy)
	}

	// Build child modules
	for _, child := range mod.Imports {
		a.buildModule(parent, child)
	}
}
