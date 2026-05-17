package lifecycle

// OnModuleInit is called when the module containing this provider is initialized.
// Runs per-module, depth-first during app startup.
type OnModuleInit interface {
	OnModuleInit() error
}

// OnApplicationBootstrap is called after all modules are initialized,
// but before the application starts serving (HTTP or signals).
// Runs once for all providers after OnModuleInit completes.
type OnApplicationBootstrap interface {
	OnApplicationBootstrap() error
}

// OnModuleDestroy is called when the module containing this provider is destroyed.
// Runs per-module, reverse depth-first during shutdown.
type OnModuleDestroy interface {
	OnModuleDestroy() error
}

// BeforeApplicationShutdown is called before shutdown begins,
// before OnApplicationShutdown. Runs once in reverse order.
// This is useful for graceful drain-stop scenarios where you need
// to stop accepting new work before closing connections.
type BeforeApplicationShutdown interface {
	BeforeApplicationShutdown() error
}

// OnApplicationShutdown is called during application shutdown,
// after BeforeApplicationShutdown, before OnModuleDestroy.
// Runs once in reverse order.
type OnApplicationShutdown interface {
	OnApplicationShutdown() error
}

// Hooks holds lifecycle hook functions for a single provider or controller.
type Hooks struct {
	OnInit           func() error
	OnBootstrap      func() error
	OnBeforeShutdown func() error
	OnDestroy        func() error
	OnShutdown       func() error
	registry         *HookRegistry // Optional: reference to registry for dynamic hook refresh
}

// Refresh pulls the latest hooks from the registry into h. Pointer receiver
// so a bare `h.Refresh()` call mutates in place — a value receiver here
// silently no-ops on a discarded return value, which the old API allowed.
func (h *Hooks) Refresh() {
	if h.registry == nil {
		return
	}
	h.OnInit = h.registry.onInit
	h.OnBootstrap = h.registry.onBootstrap
	h.OnBeforeShutdown = h.registry.beforeShutdown
	h.OnDestroy = h.registry.onDestroy
	h.OnShutdown = h.registry.onShutdown
}

// HasRegistry returns true if the hooks have an associated registry (for HookedFactory pattern).
func (h Hooks) HasRegistry() bool {
	return h.registry != nil
}

// CollectHooks checks if a value implements lifecycle interfaces
// and returns the collected hooks.
func CollectHooks(v any) Hooks {
	var hooks Hooks

	if init, ok := v.(OnModuleInit); ok {
		hooks.OnInit = init.OnModuleInit
	}
	if bootstrap, ok := v.(OnApplicationBootstrap); ok {
		hooks.OnBootstrap = bootstrap.OnApplicationBootstrap
	}
	if beforeShutdown, ok := v.(BeforeApplicationShutdown); ok {
		hooks.OnBeforeShutdown = beforeShutdown.BeforeApplicationShutdown
	}
	if destroy, ok := v.(OnModuleDestroy); ok {
		hooks.OnDestroy = destroy.OnModuleDestroy
	}
	if shutdown, ok := v.(OnApplicationShutdown); ok {
		hooks.OnShutdown = shutdown.OnApplicationShutdown
	}

	return hooks
}
