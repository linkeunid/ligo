package ligo

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
	"github.com/linkeunid/ligo/internal/http"
)

// App is the core application instance.
type App struct {
	mu           sync.Mutex
	started      bool
	modules      []module.Module
	providers    []Provider
	container    *container.Container
	moduleHooks  struct {
		onInit  [][]func() error
		onDestroy [][]func() error
	}
	opts options
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

	// Register logger as a provider for injection (as interface type)
	loggerType := reflect.TypeOf((*logger.Logger)(nil)).Elem()
	root.Register(loggerType, container.NewEntry(nil, a.opts.logger, nil, false, true))

	// Register root-level providers
	for _, p := range a.providers {
		a.registerProvider(root, p)
	}

	// Build module graph and log dependencies
	for _, mod := range a.modules {
		a.buildModule(root, mod)
		a.opts.logger.LogWithContext(logger.ContextDIContainer, fmt.Sprintf("%s module initialized", mod.Name))
	}

	a.container = root

	// Execute OnStart hooks
	for _, hook := range a.opts.onStart {
		if err := hook(nil); err != nil {
			return fmt.Errorf("OnStart hook failed: %w", err)
		}
	}

	// Execute OnModuleInit hooks
	for _, moduleHooks := range a.moduleHooks.onInit {
		for _, hook := range moduleHooks {
			if err := hook(); err != nil {
				return fmt.Errorf("OnModuleInit hook failed: %w", err)
			}
		}
	}

	// Register controllers if router is configured
	if a.opts.router != nil {
		// Set container on router for request-scoped DI
		if sc, ok := a.opts.router.(http.SetContainerRouter); ok {
			sc.SetContainer(root)
		}

		// Set logger on router if it supports it
		if sl, ok := a.opts.router.(http.SetLoggerRouter); ok {
			sl.SetLogger(a.opts.logger)
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
		if a.opts.autoPort {
			return a.serveWithRetry(a.opts.addr)
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
