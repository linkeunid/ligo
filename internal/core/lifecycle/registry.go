package lifecycle

// HookFunc represents a lifecycle hook function.
type HookFunc func() error

// HookRegistry stores lifecycle hooks with their associated type.
type HookRegistry struct {
	onInit         HookFunc
	onBootstrap    HookFunc
	beforeShutdown HookFunc
	onShutdown     HookFunc
	onDestroy      HookFunc
}

// NewHookRegistry creates a new hook registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{}
}

// OnInit sets the OnModuleInit hook.
func (r *HookRegistry) OnInit(fn HookFunc) *HookRegistry {
	r.onInit = fn
	return r
}

// OnBootstrap sets the OnApplicationBootstrap hook.
func (r *HookRegistry) OnBootstrap(fn HookFunc) *HookRegistry {
	r.onBootstrap = fn
	return r
}

// BeforeShutdown sets the BeforeApplicationShutdown hook.
func (r *HookRegistry) BeforeShutdown(fn HookFunc) *HookRegistry {
	r.beforeShutdown = fn
	return r
}

// OnShutdown sets the OnApplicationShutdown hook.
func (r *HookRegistry) OnShutdown(fn HookFunc) *HookRegistry {
	r.onShutdown = fn
	return r
}

// OnDestroy sets the OnModuleDestroy hook.
func (r *HookRegistry) OnDestroy(fn HookFunc) *HookRegistry {
	r.onDestroy = fn
	return r
}

// Registerable is the interface that services can implement to explicitly
// register their lifecycle hooks. This enables compile-time safety via method
// expressions instead of relying on duck typing.
//
// Example:
//
//	type Database struct {}
//
//	func (d *Database) Connect() error { ... }
//	func (d *Database) Close() error { ... }
//
//	func (d *Database) Register(r *lifecycle.HookRegistry) {
//	    r.OnInit(d.Connect)      // Method expression - compile-time checked
//	    r.OnShutdown(d.Close)    // If Close doesn't exist → compile error
//	}
type Registerable interface {
	Register(*HookRegistry)
}

// RegisterFrom calls the Register method on the target if it implements Registerable.
// This enables services to explicitly register their hooks with compile-time safety.
func (r *HookRegistry) RegisterFrom(target any) {
	if reg, ok := target.(Registerable); ok {
		reg.Register(r)
	}
}

// ToHooks converts the registry to a Hooks struct.
// The registry reference is stored so hooks can be refreshed after RegisterFrom is called.
func (r *HookRegistry) ToHooks() Hooks {
	return Hooks{
		OnInit:           r.onInit,
		OnBootstrap:      r.onBootstrap,
		OnBeforeShutdown: r.beforeShutdown,
		OnDestroy:        r.onDestroy,
		OnShutdown:       r.onShutdown,
		registry:         r,
	}
}

// GetInitHooks returns the collected OnInit hooks for module-level use.
func (r *ModuleHookRegistry) GetInitHooks() []func() error {
	return r.onInit
}

// GetDestroyHooks returns the destroy hooks as a slice (for module-level use).
func (r *ModuleHookRegistry) GetDestroyHooks() []func() error {
	return r.onDestroy
}

// ModuleHookRegistry stores module-level lifecycle hooks.
type ModuleHookRegistry struct {
	onInit    []func() error
	onDestroy []func() error
}

// NewModuleHookRegistry creates a new module hook registry.
func NewModuleHookRegistry() *ModuleHookRegistry {
	return &ModuleHookRegistry{}
}

// OnInit adds a module init hook.
func (r *ModuleHookRegistry) OnInit(fn func() error) *ModuleHookRegistry {
	r.onInit = append(r.onInit, fn)
	return r
}

// OnDestroy adds a module destroy hook.
func (r *ModuleHookRegistry) OnDestroy(fn func() error) *ModuleHookRegistry {
	r.onDestroy = append(r.onDestroy, fn)
	return r
}
