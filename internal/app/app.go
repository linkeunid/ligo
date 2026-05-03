package app

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"

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

// BuildProviderEntry builds a container entry from a provider and returns its lifecycle hooks.
func BuildProviderEntry(p Provider) (container.ProviderEntry, ProviderHooks) {
	if p.Eager() != nil {
		hooks := collectProviderHooks(p.Eager())
		return container.NewEntry(nil, p.Eager(), nil, p.IsTransient(), p.IsExported()), hooks
	}

	fn := reflect.ValueOf(p).MethodByName("Fn").Call([]reflect.Value{})[0].Interface()
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	argTypes := make([]reflect.Type, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		argTypes[i] = fnType.In(i)
	}

	entry := container.NewEntry(func(args []reflect.Value) (any, error) {
		out := fnValue.Call(args)
		if len(out) == 0 {
			return nil, fmt.Errorf("ligo: factory function must return a value")
		}
		return out[0].Interface(), nil
	}, nil, argTypes, p.IsTransient(), p.IsExported())

	return entry, ProviderHooks{} // Eager providers have hooks, factories don't know yet
}

// RegisterProvider registers a provider in the container and returns its lifecycle hooks.
func RegisterProvider(c *container.Container, p Provider) ProviderHooks {
	entry, hooks := BuildProviderEntry(p)
	c.Register(p.Type(), entry)
	return hooks
}

// BuildModule registers providers from a module and its imports in the container.
// The module must have been pre-expanded by ExpandModule — dynamic fields are not processed here.
func BuildModule(parent *container.Container, mod module.Module, hooks *ModuleHooks) {
	modContainer := parent

	// Pre-allocate capacity for provider hooks to reduce slice growth
	if cap(hooks.Providers) < len(hooks.Providers)+len(mod.Providers) {
		newCap := len(hooks.Providers) + len(mod.Providers)
		newSlice := make([]ProviderHooks, len(hooks.Providers), newCap)
		copy(newSlice, hooks.Providers)
		hooks.Providers = newSlice
	}

	for _, p := range mod.Providers {
		provider, _ := p.(Provider)
		entry, providerHooks := BuildProviderEntry(provider)

		if provider.IsExported() {
			parent.Register(provider.Type(), entry)
		} else {
			modContainer.Register(provider.Type(), entry)
		}

		// Collect hooks if any are implemented
		if providerHooks.OnInit != nil || providerHooks.OnBootstrap != nil || providerHooks.OnDestroy != nil || providerHooks.OnShutdown != nil {
			hooks.Providers = append(hooks.Providers, providerHooks)
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
	Providers []ProviderHooks // NEW: provider-level hooks
}

// ProviderHooks holds lifecycle hooks for a single provider instance.
type ProviderHooks struct {
	OnInit      func() error
	OnBootstrap func() error
	OnDestroy   func() error
	OnShutdown  func() error
}

// collectProviderHooks checks if a value implements lifecycle interfaces
// and returns the collected hooks.
func collectProviderHooks(v any) ProviderHooks {
	var hooks ProviderHooks

	if init, ok := v.(interface{ OnModuleInit() error }); ok {
		hooks.OnInit = init.OnModuleInit
	}
	if bootstrap, ok := v.(interface{ OnApplicationBootstrap() error }); ok {
		hooks.OnBootstrap = bootstrap.OnApplicationBootstrap
	}
	if destroy, ok := v.(interface{ OnModuleDestroy() error }); ok {
		hooks.OnDestroy = destroy.OnModuleDestroy
	}
	if shutdown, ok := v.(interface{ OnApplicationShutdown() error }); ok {
		hooks.OnShutdown = shutdown.OnApplicationShutdown
	}

	return hooks
}

// ExpandModule materializes a dynamic module and recursively expands its imports,
// deduplicating by module name using visited. Returns (expanded, true) if the module
// should be processed, or (zero, false) if it was already visited.
func ExpandModule(mod module.Module, visited map[string]bool) (module.Module, bool) {
	if visited[mod.Name] {
		return module.Module{}, false
	}
	visited[mod.Name] = true

	if mod.Dynamic != nil {
		dynamic := mod.Dynamic.Factory(mod.Dynamic.Options...)
		mod.Providers = append(mod.Providers, dynamic.Providers...)
		mod.Controllers = append(mod.Controllers, dynamic.Controllers...)
		mod.Imports = append(mod.Imports, dynamic.Imports...)
		mod.Middlewares = append(mod.Middlewares, dynamic.Middlewares...)
		mod.OnInit = append(mod.OnInit, dynamic.OnInit...)
		mod.OnDestroy = append(mod.OnDestroy, dynamic.OnDestroy...)
		mod.Dynamic = nil
	}

	var expandedImports []module.Module
	for _, child := range mod.Imports {
		if expanded, ok := ExpandModule(child, visited); ok {
			expandedImports = append(expandedImports, expanded)
		}
	}
	mod.Imports = expandedImports

	return mod, true
}

// ExpandModules expands and deduplicates a slice of top-level modules.
// Returns nil if modules is nil or empty.
func ExpandModules(modules []module.Module) []module.Module {
	visited := make(map[string]bool)
	var result []module.Module
	for _, mod := range modules {
		if expanded, ok := ExpandModule(mod, visited); ok {
			result = append(result, expanded)
		}
	}
	return result
}

// WaitForShutdown waits for SIGINT/SIGTERM signals in non-HTTP mode.
// This enables non-HTTP applications (bots, CLI runners) to block until
// shutdown signals instead of returning immediately from Run().
func WaitForShutdown(log logger.Logger) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	log.Info("Ligo application running (non-HTTP mode). Press Ctrl+C to stop.",
		logger.Field{Key: "context", Value: logger.ContextLifecycle})

	received := <-sig
	log.Info("Shutdown signal received",
		logger.Field{Key: "signal", Value: received.String()},
		logger.Field{Key: "context", Value: logger.ContextLifecycle})

	return nil
}
