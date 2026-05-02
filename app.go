package ligo

// Package ligo provides a modular Go framework with lightweight dependency injection,
// inspired by NestJS. It offers HTTP routing with an adapter pattern, a powerful DI container,
// module system, and request processing with Guards, Pipes, Interceptors, and Exception Filters.

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/linkeunid/ligo/internal/app"
	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
	"github.com/linkeunid/ligo/internal/http"
)

// App represents a Ligo application with dependency injection, module management,
// and HTTP server capabilities.
type App struct {
	mu           sync.Mutex
	started      bool
	modules      []module.Module
	providers    []Provider
	container    *container.Container
	moduleHooks  *app.ModuleHooks
	opts         options
}

// New creates a new Ligo application with the given options.
// Options include WithRouter, WithAddr, WithMiddleware, OnStart, and OnStop.
//
// Example:
//
//	app := ligo.New(
//	    ligo.WithRouter(echo.NewAdapter()),
//	    ligo.WithAddr(":8080"),
//	)
func New(opts ...Option) *App {
	op := defaultOptions()
	for _, opt := range opts {
		opt(&op)
	}
	return &App{
		opts:        op,
		moduleHooks: &app.ModuleHooks{},
	}
}

// Register registers one or more modules with the application.
// Modules must be registered before calling Run().
// Panics if called after the application has started.
//
// Example:
//
//	app.Register(
//	    user.Module(),
//	    auth.Module(),
//	)
func (a *App) Register(modules ...module.Module) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.started {
		panic(&ErrAppAlreadyStarted{})
	}

	a.modules = append(a.modules, modules...)
}

// Provide registers global providers that are available across all modules.
// Providers must be registered before calling Run().
// Panics if called after the application has started.
//
// Example:
//
//	app.Provide(
//	    ligo.Value("config-value"),
//	    ligo.Factory[*Config](NewConfig),
//	)
func (a *App) Provide(providers ...Provider) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.started {
		panic(&ErrAppAlreadyStarted{})
	}

	a.providers = append(a.providers, providers...)
}

// Run starts the HTTP server and blocks until the server is shut down.
// It builds the DI container, registers all modules and providers,
// executes OnModuleInit hooks, starts the server, and waits for shutdown.
// On shutdown, it executes OnModuleDestroy and OnStop hooks.
//
// Example:
//
//	if err := app.Run(); err != nil {
//	    log.Fatal(err)
//	}
func (a *App) Run() error {
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		panic(&ErrAppAlreadyStarted{})
	}
	a.started = true
	a.mu.Unlock()

	a.opts.logger.Info("Starting ligo application", logger.Field{Key: "context", Value: logger.ContextApp})

	root := container.New()

	loggerType := reflect.TypeOf((*logger.Logger)(nil)).Elem()
	root.Register(loggerType, container.NewEntry(nil, a.opts.logger, nil, false, true))

	for _, p := range a.providers {
		app.RegisterProvider(root, p)
	}

	expandedModules := app.ExpandModules(a.modules)

	for _, mod := range expandedModules {
		app.BuildModule(root, mod, a.moduleHooks)
		a.opts.logger.LogWithContext(logger.ContextDIContainer, fmt.Sprintf("%s module initialized", mod.Name))
	}

	a.container = root

	for _, hook := range a.opts.onStart {
		if err := hook(nil); err != nil {
			return fmt.Errorf("OnStart hook failed: %w", err)
		}
	}

	if err := app.ExecuteHooks(a.moduleHooks.OnInit, a.opts.logger, "OnModuleInit"); err != nil {
		return err
	}

	if a.opts.router != nil {
		if sc, ok := a.opts.router.(http.SetContainerRouter); ok {
			sc.SetContainer(root)
		}

		if sl, ok := a.opts.router.(http.SetLoggerRouter); ok {
			sl.SetLogger(a.opts.logger)
		}

		binder := http.NewBinder(a.container, a.opts.router, a.opts.logger)

		for _, mw := range a.opts.middlewares {
			a.opts.router.Use(mw)
		}

		if err := binder.BindControllers(expandedModules); err != nil {
			return err
		}
	}

	a.opts.logger.Info("Ligo application started",
		logger.Field{Key: "context", Value: logger.ContextApp},
		logger.Field{Key: "addr", Value: a.opts.addr},
	)

	if a.opts.router != nil {
		onStop := make([]func(any) error, len(a.opts.onStop))
		for i, h := range a.opts.onStop {
			onStop[i] = h
		}
		return app.ServeWithRetry(app.ServeOptions{
			Router:          a.opts.router,
			Logger:          a.opts.logger,
			Addr:            a.opts.addr,
			AutoPort:        a.opts.autoPort,
			GracefulTimeout: a.opts.gracefulTimeout,
			ModuleHooks:     a.moduleHooks,
			OnStop:          onStop,
		})
	}
	return nil
}

func (a *App) Container() *container.Container {
	if a.container == nil {
		panic("ligo: cannot access container before Run()")
	}
	return a.container
}
