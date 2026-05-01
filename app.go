package ligo

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
	"github.com/linkeunid/ligo/internal/http"
)

// App is the core application instance.
type App struct {
	mu        sync.Mutex
	started   bool
	modules   []module.Module
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
func (a *App) Register(modules ...module.Module) {
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

	a.opts.logger.Info("Starting ligo application", logger.Field{Key: "context", Value: logger.ContextApp})

	// Build root container
	root := container.New()

	// Register root-level providers
	for _, p := range a.providers {
		a.registerProvider(root, p)
	}

	// Build module graph and log dependencies
	for _, mod := range a.modules {
		a.buildModule(root, mod, a.opts.logger)
		a.opts.logger.LogWithContext(logger.ContextDIContainer, fmt.Sprintf("%s module initialized", mod.Name))
	}

	a.container = root

	// Register controllers if router is configured
	if a.opts.router != nil {
		// Set logger on router if it supports it
		if echoAdapter, ok := a.opts.router.(interface{ SetLogger(logger.Logger) }); ok {
			echoAdapter.SetLogger(a.opts.logger)
		}

		// Build binder
		binder := http.NewBinder(a.container, a.opts.router, a.opts.logger)

		// Apply global middleware
		for _, mw := range a.opts.middlewares {
			a.opts.router.Use(mw)
		}

		// Bind all controllers with module middleware
		if err := binder.BindControllers(a.modules); err != nil {
			return err
		}
	}

	a.opts.logger.Info("Ligo application started",
		logger.Field{Key: "context", Value: logger.ContextApp},
		logger.Field{Key: "addr", Value: a.opts.addr},
	)

	if a.opts.router != nil {
		if a.opts.gracefulShutdown {
			return a.runWithGracefulShutdown()
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

// runWithGracefulShutdown runs the server with graceful shutdown on SIGINT/SIGTERM.
func (a *App) runWithGracefulShutdown() error {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)

	go func() {
		errChan <- a.opts.router.Serve(a.opts.addr)
	}()

	select {
	case <-shutdownChan:
		a.opts.logger.Info("Shutting down gracefully...", logger.Field{Key: "context", Value: logger.ContextLifecycle})

		ctx, cancel := context.WithTimeout(context.Background(), a.opts.gracefulTimeout)
		defer cancel()

		for _, hook := range a.opts.onStop {
			if err := hook(ctx); err != nil {
				a.opts.logger.Error("OnStop hook failed", logger.Field{Key: "error", Value: err})
			}
		}

		if gs, ok := a.opts.router.(http.GracefulServer); ok {
			if err := gs.Shutdown(ctx); err != nil {
				return err
			}
		}
		return nil
	case err := <-errChan:
		return err
	}
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

func (a *App) buildModule(parent *container.Container, mod module.Module, log Logger) {
	modContainer := parent // flat graph - modules share root container

	// Register module providers
	for _, p := range mod.Providers {
		provider := p.(Provider)
		name := logger.ExtractProviderName(provider.Fn())
		if name == "unknown" && provider.Eager() != nil {
			name = logger.ExtractProviderName(provider.Eager())
		}
		if provider.IsExported() {
			a.registerProvider(parent, provider)
		} else {
			a.registerProvider(modContainer, provider)
		}
	}

	// Build child modules
	for _, child := range mod.Imports {
		a.buildModule(parent, child, log)
	}
}