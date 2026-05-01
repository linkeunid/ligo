package ligo

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/linkeunid/ligo/internal/container"
)

// App is the core application instance.
type App struct {
	mu        sync.Mutex
	started   bool
	modules   []Module
	providers []Provider
	container *container.Container
	opts      options
}

// New creates a new Ligo application.
func New(opts ...Option) *App {
	op := defaultOptions()
	for _, opt := range opts {
		opt(&op)
	}
	return &App{
		opts: op,
	}
}

// Register adds modules to the application.
func (a *App) Register(modules ...Module) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.started {
		panic(&ErrAppAlreadyStarted{})
	}

	a.modules = append(a.modules, modules...)
}

// Provide registers ad-hoc providers at the root level.
func (a *App) Provide(providers ...Provider) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.started {
		panic(&ErrAppAlreadyStarted{})
	}

	a.providers = append(a.providers, providers...)
}

// Run builds the DI container, resolves all providers, and starts the server.
func (a *App) Run() error {
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		panic(&ErrAppAlreadyStarted{})
	}
	a.started = true
	a.mu.Unlock()

	// Build root container
	root := container.New()

	// Register root-level providers
	for _, p := range a.providers {
		a.registerProvider(root, p)
	}

	// Build module graph
	for _, mod := range a.modules {
		a.buildModule(root, mod)
	}

	a.container = root

	// Register controllers if router is configured
	if a.opts.router != nil {
		if err := a.registerControllers(); err != nil {
			return err
		}
		return a.opts.router.Serve(a.opts.addr)
	}

	return nil
}

// Container returns the internal container (escape hatch).
// Panics if called before Run().
func (a *App) Container() *container.Container {
	if a.container == nil {
		panic("ligo: cannot access container before Run()")
	}
	return a.container
}

func (a *App) registerProvider(c *container.Container, p Provider) {
	entry := a.buildProviderEntry(p)
	c.Register(p.Type(), entry)
}

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

func (a *App) buildModule(parent *container.Container, mod Module) {
	// Create module container with parent
	modContainer := parent // flat graph - modules share root container

	// Register module providers
	for _, p := range mod.providers {
		if p.exported {
			a.registerProvider(parent, p) // exported to root
		} else {
			a.registerProvider(modContainer, p)
		}
	}

	// Build child modules
	for _, child := range mod.imports {
		a.buildModule(parent, child)
	}
}

func (a *App) registerControllers() error {
	for _, mod := range a.modules {
		for _, cc := range mod.controllers {
			// Resolve dependencies from container
			args := make([]reflect.Value, len(cc.argTypes))
			for i, argType := range cc.argTypes {
				resolved := container.ResolveByType(a.container, argType)
				if resolved == nil {
					return fmt.Errorf("ligo: missing dependency %s for controller", argType.String())
				}
				args[i] = reflect.ValueOf(resolved)
			}

			// Call constructor with resolved dependencies
			fnValue := reflect.ValueOf(cc.fn)
			out := fnValue.Call(args)
			if len(out) == 0 {
				return fmt.Errorf("ligo: controller constructor must return a Controller")
			}
			ctrl, ok := out[0].Interface().(Controller)
			if !ok {
				return fmt.Errorf("ligo: constructor must return Controller")
			}
			if ctrl != nil {
				ctrl.Routes(a.opts.router)
			}
		}
	}
	return nil
}
