package app

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/lifecycle"
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
	Hooks() *lifecycle.HookRegistry
}

// BuildProviderEntry builds a container entry from a provider and returns its lifecycle hooks.
func BuildProviderEntry(p Provider) (container.ProviderEntry, lifecycle.Hooks) {
	// Start with explicit hooks if registered
	var hooks lifecycle.Hooks
	if registry := p.Hooks(); registry != nil {
		hooks = registry.ToHooks()
	}

	if p.Eager() != nil {
		// For eager providers, call RegisterFrom if explicit registry exists
		// This allows services to explicitly register their hooks with compile-time safety
		if registry := p.Hooks(); registry != nil {
			registry.RegisterFrom(p.Eager())
			hooks = registry.ToHooks()
		}

		// Also collect interface-based hooks for backward compatibility
		// Explicit hooks take precedence over interface-based hooks
		if hooks.OnInit == nil || hooks.OnBootstrap == nil || hooks.OnBeforeShutdown == nil || hooks.OnDestroy == nil || hooks.OnShutdown == nil {
			interfaceHooks := lifecycle.CollectHooks(p.Eager())
			if hooks.OnInit == nil {
				hooks.OnInit = interfaceHooks.OnInit
			}
			if hooks.OnBootstrap == nil {
				hooks.OnBootstrap = interfaceHooks.OnBootstrap
			}
			if hooks.OnBeforeShutdown == nil {
				hooks.OnBeforeShutdown = interfaceHooks.OnBeforeShutdown
			}
			if hooks.OnDestroy == nil {
				hooks.OnDestroy = interfaceHooks.OnDestroy
			}
			if hooks.OnShutdown == nil {
				hooks.OnShutdown = interfaceHooks.OnShutdown
			}
		}
		return container.NewEntry(nil, p.Eager(), nil, p.IsTransient(), p.IsExported(), nil), hooks
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
	}, nil, argTypes, p.IsTransient(), p.IsExported(), p.Hooks())

	return entry, hooks // Explicit hooks are stored for factory providers
}

// RegisterProvider registers a provider in the container and returns its lifecycle hooks.
func RegisterProvider(c *container.Container, p Provider) lifecycle.Hooks {
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
		newSlice := make([]lifecycle.Hooks, len(hooks.Providers), newCap)
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

		// Collect hooks if any are implemented or if registry is set (for HookedFactory pattern)
		if providerHooks.OnInit != nil || providerHooks.OnBootstrap != nil || providerHooks.OnBeforeShutdown != nil || providerHooks.OnDestroy != nil || providerHooks.OnShutdown != nil || providerHooks.HasRegistry() {
			hooks.Providers = append(hooks.Providers, providerHooks)
		}
	}

	if len(mod.OnInit) > 0 {
		hooks.OnInit = append(hooks.OnInit, mod.OnInit)
	}
	if len(mod.OnDestroy) > 0 {
		hooks.OnDestroy = append(hooks.OnDestroy, mod.OnDestroy)
	}

	// Handle explicit module hooks registry
	if mod.Hooks != nil {
		if initHooks := mod.Hooks.GetInitHooks(); len(initHooks) > 0 {
			hooks.OnInit = append(hooks.OnInit, initHooks)
		}
		if destroyHooks := mod.Hooks.GetDestroyHooks(); len(destroyHooks) > 0 {
			hooks.OnDestroy = append(hooks.OnDestroy, destroyHooks)
		}
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
	Providers []lifecycle.Hooks // provider/controller-level hooks
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
