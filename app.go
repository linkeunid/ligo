package ligo

// Package ligo provides a modular Go framework with lightweight dependency injection,
// inspired by NestJS. It offers HTTP routing with an adapter pattern, a powerful DI container,
// module system, and request processing with Guards, Pipes, Interceptors, and Exception Filters.

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/linkeunid/ligo/internal/app"
	"github.com/linkeunid/ligo/internal/core/lifecycle"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
	"github.com/linkeunid/ligo/internal/di"
	"github.com/linkeunid/ligo/internal/http"
)

// App represents a Ligo application with dependency injection, module management,
// and HTTP server capabilities.
type App struct {
	mu        sync.Mutex
	started   bool
	modules   []module.Module
	providers []Provider
	// container is written by Run (which may run in a goroutine while a
	// test reads from Container()), so it must be published atomically.
	container   atomic.Pointer[di.Container]
	moduleHooks *app.ModuleHooks
	opts        options
}

// hookTask represents a single hook execution task for parallel processing.
type hookTask struct {
	provider *lifecycle.Hooks
	hook     func() error
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
	if err := a.ensureNotStarted(); err != nil {
		return err
	}

	a.opts.logger.Info("Starting ligo application", logger.Field{Key: "context", Value: logger.ContextApp})

	root := a.buildContainer()
	a.container.Store(root)

	expandedModules := app.ExpandModules(a.modules)
	a.initializeModules(root, expandedModules)

	router := a.setupRouter(root)
	controllerHooks, err := a.bindControllers(root, router, expandedModules)
	if err != nil {
		return err
	}
	a.moduleHooks.Providers = append(a.moduleHooks.Providers, controllerHooks...)

	if err := a.resolveEagerProviders(root); err != nil {
		return err
	}

	if err := a.executeStartupHooks(); err != nil {
		return err
	}

	a.opts.logger.Info("Ligo application started", a.getLogFields()...)

	return a.startServer()
}

// ensureNotStarted checks if the app has already started and panics if so.
func (a *App) ensureNotStarted() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.started {
		return &ErrAppAlreadyStarted{}
	}
	a.started = true
	return nil
}

// buildContainer creates and configures the DI container with all providers.
func (a *App) buildContainer() *di.Container {
	root := di.New(a.opts.logger)

	loggerType := reflect.TypeFor[logger.Logger]()
	root.Register(loggerType, di.NewEntry(nil, a.opts.logger, nil, false, true, nil))

	for _, p := range a.providers {
		app.RegisterProvider(root, p)
	}

	return root
}

// initializeModules registers all modules with the di.
func (a *App) initializeModules(root *di.Container, expandedModules []module.Module) {
	for _, mod := range expandedModules {
		app.BuildModule(root, mod, a.moduleHooks)
		a.opts.logger.LogWithContext(logger.ContextDIContainer, fmt.Sprintf("%s module initialized", mod.Name))
	}
}

// setupRouter configures the HTTP router with the container and middleware.
func (a *App) setupRouter(root *di.Container) http.Router {
	var router http.Router
	if a.opts.router != nil {
		router = a.opts.router
		if sc, ok := router.(http.SetContainerRouter); ok {
			sc.SetContainer(root)
		}
		if sl, ok := router.(http.SetLoggerRouter); ok {
			sl.SetLogger(a.opts.logger)
		}
		for _, mw := range a.opts.middlewares {
			router.Use(mw)
		}
	} else {
		router = &http.NullRouter{}
	}
	return router
}

// bindControllers binds all controllers and collects their lifecycle hooks.
func (a *App) bindControllers(root *di.Container, router http.Router, expandedModules []module.Module) ([]lifecycle.Hooks, error) {
	binder := http.NewBinder(root, router, a.opts.logger)
	controllerHooks, err := binder.BindControllers(expandedModules)
	if err != nil {
		return nil, err
	}
	return controllerHooks, nil
}

// resolveEagerProviders instantiates providers flagged via HookedSingleton
// even when no other provider depends on them. This forces RegisterFrom to
// run so their explicit hooks attach before OnInit / OnBootstrap fire.
func (a *App) resolveEagerProviders(root *di.Container) error {
	for _, typ := range a.moduleHooks.EagerTypes {
		if _, err := di.ResolveByType(root, typ); err != nil {
			return fmt.Errorf("ligo: eager resolve %s: %w", typ, err)
		}
	}
	return nil
}

// executeStartupHooks runs OnStart, OnInit, and OnBootstrap hooks.
func (a *App) executeStartupHooks() error {
	// Run custom OnStart hooks
	for _, hook := range a.opts.onStart {
		if err := hook(nil); err != nil {
			return fmt.Errorf("OnStart hook failed: %w", err)
		}
	}

	// Run module OnInit hooks
	if err := app.ExecuteHooks(a.moduleHooks.OnInit, a.opts.logger, "OnModuleInit"); err != nil {
		return err
	}

	// Run provider OnInit hooks
	if err := a.executeProviderHooks(func(hooks *lifecycle.Hooks) func() error {
		return hooks.OnInit
	}); err != nil {
		return err
	}

	// Run provider OnBootstrap hooks
	return a.executeProviderHooks(func(hooks *lifecycle.Hooks) func() error {
		return hooks.OnBootstrap
	})
}

// executeProviderHooks executes a specific hook type across all providers.
// Sequential in registration order by default; parallel when the app opted in
// via WithParallelHooks().
func (a *App) executeProviderHooks(getHook func(*lifecycle.Hooks) func() error) error {
	if a.opts.parallelHooks {
		return executeProviderHooksParallel(a.moduleHooks.Providers, getHook)
	}
	return executeProviderHooksSequential(a.moduleHooks.Providers, getHook)
}

// executeProviderHooksSequential runs each provider hook in registration order.
// Stops at the first error and returns it (with provider context).
func executeProviderHooksSequential(providers []lifecycle.Hooks, getHook func(*lifecycle.Hooks) func() error) error {
	for i := range providers {
		if providers[i].HasRegistry() {
			providers[i].Refresh()
		}
		hook := getHook(&providers[i])
		if hook == nil {
			continue
		}
		if err := hook(); err != nil {
			return fmt.Errorf("provider hook failed: %w", err)
		}
	}
	return nil
}

// executeProviderHooksParallel executes provider hooks in parallel where possible.
// Hooks that depend on shared state are executed sequentially.
func executeProviderHooksParallel(providers []lifecycle.Hooks, getHook func(*lifecycle.Hooks) func() error) error {
	var tasks []hookTask
	for i := range providers {
		if providers[i].HasRegistry() {
			providers[i].Refresh()
		}
		if hook := getHook(&providers[i]); hook != nil {
			tasks = append(tasks, hookTask{provider: &providers[i], hook: hook})
		}
	}

	// Execute hooks in parallel using errgroup
	return executeHooksParallel(tasks)
}

// executeHooksParallel executes hooks in parallel and returns errors.Join
// of all hook failures (nil if all succeeded).
func executeHooksParallel(tasks []hookTask) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(tasks))

	for _, task := range tasks {
		wg.Go(func() {
			if err := task.hook(); err != nil {
				errChan <- err
			}
		})
	}

	wg.Wait()
	close(errChan)

	var results []error
	for err := range errChan {
		results = append(results, err)
	}
	return errors.Join(results...)
}

// getLogFields returns log fields for application startup.
func (a *App) getLogFields() []logger.Field {
	fields := []logger.Field{{Key: "context", Value: logger.ContextApp}}
	if a.opts.router != nil {
		fields = append(fields, logger.Field{Key: "addr", Value: a.opts.addr})
	}
	return fields
}

// startServer starts the HTTP server or waits for shutdown signals in non-HTTP mode.
func (a *App) startServer() error {
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
			AppShutdown:     a.shutdown,
		})
	} else {
		// Non-HTTP mode: wait for shutdown signals
		if err := app.WaitForShutdown(a.opts.logger); err != nil {
			return err
		}
		return a.shutdown()
	}
}

// shutdown executes BeforeApplicationShutdown, OnApplicationShutdown, and OnModuleDestroy hooks in reverse order.
// Logs each error as it happens and returns errors.Join of every failure so callers
// can propagate a non-zero exit status.
func (a *App) shutdown() error {
	var errs []error
	for i := len(a.moduleHooks.Providers) - 1; i >= 0; i-- {
		// Only refresh if registry exists (HookedFactory pattern where RegisterFrom may have been called during resolution)
		if a.moduleHooks.Providers[i].HasRegistry() {
			a.moduleHooks.Providers[i].Refresh()
		}
		if h := a.moduleHooks.Providers[i].OnBeforeShutdown; h != nil {
			if err := h(); err != nil {
				a.opts.logger.Error("BeforeApplicationShutdown hook failed", logger.Field{Key: "error", Value: err})
				errs = append(errs, fmt.Errorf("BeforeApplicationShutdown: %w", err))
			}
		}
		if h := a.moduleHooks.Providers[i].OnShutdown; h != nil {
			if err := h(); err != nil {
				a.opts.logger.Error("OnApplicationShutdown hook failed", logger.Field{Key: "error", Value: err})
				errs = append(errs, fmt.Errorf("OnApplicationShutdown: %w", err))
			}
		}
		if h := a.moduleHooks.Providers[i].OnDestroy; h != nil {
			if err := h(); err != nil {
				a.opts.logger.Error("OnModuleDestroy hook failed", logger.Field{Key: "error", Value: err})
				errs = append(errs, fmt.Errorf("OnModuleDestroy: %w", err))
			}
		}
	}

	if err := app.ExecuteHooks(a.moduleHooks.OnDestroy, a.opts.logger, "OnModuleDestroy"); err != nil {
		a.opts.logger.Error("Module OnDestroy hooks failed", logger.Field{Key: "error", Value: err})
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (a *App) Container() *di.Container {
	c := a.container.Load()
	if c == nil {
		panic("ligo: cannot access container before Run()")
	}
	return c
}

// Resolve returns an instance of type T from the application container.
// Returns the zero value and an error when the type cannot be resolved
// (missing provider, ambiguous interface, circular dependency, factory error).
// Must be called after Run() has built the container.
//
// Example:
//
//	user, err := ligo.Resolve[*UserService](app)
//	if err != nil { /* handle */ }
func Resolve[T any](a *App) (T, error) {
	return di.Resolve[T](a.Container())
}

// MustResolve returns an instance of type T from the application container.
// Panics on any resolution failure. Use Resolve when you can handle the error.
// Must be called after Run() has built the container.
func MustResolve[T any](a *App) T {
	return di.MustResolve[T](a.Container())
}
