package app

import (
	"fmt"
	"reflect"

	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
)

// Provider is the interface for dependency providers (re-exported from root package).
// We use interface{} here to avoid circular import; the root package will type-assert.
type Provider interface {
	Type() reflect.Type
	Eager() any
	IsTransient() bool
	IsExported() bool
}

// BuildProviderEntry builds a container entry from a provider.
func BuildProviderEntry(p Provider) container.ProviderEntry {
	if p.Eager() != nil {
		return container.NewEntry(nil, p.Eager(), nil, p.IsTransient(), p.IsExported())
	}

	fn := reflect.ValueOf(p).MethodByName("Fn").Call([]reflect.Value{})[0].Interface()
	fnValue := reflect.ValueOf(fn)
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
	}, nil, argTypes, p.IsTransient(), p.IsExported())
}

// RegisterProvider registers a provider in the container.
func RegisterProvider(c *container.Container, p Provider) {
	entry := BuildProviderEntry(p)
	c.Register(p.Type(), entry)
}

// BuildModule registers providers from a module and its imports in the container.
func BuildModule(parent *container.Container, mod module.Module, hooks *ModuleHooks) {
	if mod.Dynamic != nil {
		dynamicMod := mod.Dynamic.Factory(mod.Dynamic.Options...)
		mod.Providers = append(mod.Providers, dynamicMod.Providers...)
		mod.Controllers = append(mod.Controllers, dynamicMod.Controllers...)
		mod.Imports = append(mod.Imports, dynamicMod.Imports...)
		mod.Middlewares = append(mod.Middlewares, dynamicMod.Middlewares...)
		mod.OnInit = append(mod.OnInit, dynamicMod.OnInit...)
		mod.OnDestroy = append(mod.OnDestroy, dynamicMod.OnDestroy...)
	}

	modContainer := parent

	for _, p := range mod.Providers {
		provider, _ := p.(Provider)
		if provider.IsExported() {
			RegisterProvider(parent, provider)
		} else {
			RegisterProvider(modContainer, provider)
		}
	}

	if len(mod.OnInit) > 0 {
		hooks.OnInit = append(hooks.OnInit, mod.OnInit)
	}
	if len(mod.OnDestroy) > 0 {
		hooks.OnDestroy = append(hooks.OnDestroy, mod.OnDestroy)
	}

	for _, child := range mod.Imports {
		BuildModule(parent, child, hooks)
	}
}

// ExecuteHooks executes module init hooks.
func ExecuteHooks(hooks [][]func() error, log logger.Logger, hookName string) error {
	for i, moduleHooks := range hooks {
		for j, hook := range moduleHooks {
			if err := hook(); err != nil {
				if log != nil {
					log.Error(fmt.Sprintf("%s hook failed (module %d, hook %d)", hookName, i, j), logger.Field{Key: "error", Value: err})
				}
				return fmt.Errorf("%s hook failed: %w", hookName, err)
			}
		}
	}
	return nil
}

// ModuleHooks holds module lifecycle hooks.
type ModuleHooks struct {
	OnInit    [][]func() error
	OnDestroy [][]func() error
}
