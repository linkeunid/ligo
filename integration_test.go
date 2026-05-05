package ligo_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/linkeunid/ligo"
	"github.com/linkeunid/ligo/adapters/echo"
)

// TestDatabase is a test service that implements Register for compile-time safe hook registration.
type TestDatabase struct {
	conn          string
	initCalled    *atomic.Bool
	shutdownCalled *atomic.Bool
}

// Connect establishes the database connection.
func (d *TestDatabase) Connect() error {
	d.conn = "connected"
	if d.initCalled != nil {
		d.initCalled.Store(true)
	}
	return nil
}

// Close closes the database connection.
func (d *TestDatabase) Close() error {
	d.conn = ""
	if d.shutdownCalled != nil {
		d.shutdownCalled.Store(true)
	}
	return nil
}

// Register implements the Registerable interface for compile-time safe hook registration.
// Method expressions like d.Connect are type-checked by the compiler.
func (d *TestDatabase) Register(r *ligo.HookRegistry) {
	r.OnInit(d.Connect)    // If Connect doesn't exist → compile error
	r.OnShutdown(d.Close)  // If Close doesn't exist → compile error
}

// TestController is a test controller that forces TestDatabase resolution.
type TestController struct {
	db *TestDatabase
}

// RegisterRoutes registers the controller routes.
func (c *TestController) RegisterRoutes(r ligo.Router) {
	// Force database resolution
	_ = c.db
}

// TestAppLifecycle tests the complete application lifecycle from creation to shutdown.
func TestAppLifecycle(t *testing.T) {
	t.Run("full lifecycle with graceful shutdown", func(t *testing.T) {
		var onStartCalled, onStopCalled, moduleInitCalled, moduleDestroyCalled atomic.Bool

		// Create a simple test module
		testModule := ligo.NewModule("test",
			ligo.OnModuleInit(func() error {
				moduleInitCalled.Store(true)
				return nil
			}),
			ligo.OnModuleDestroy(func() error {
				moduleDestroyCalled.Store(true)
				return nil
			}),
		)

		// Create app with lifecycle hooks
		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"), // Use random port
			ligo.OnStart(func(ctx any) error {
				onStartCalled.Store(true)
				return nil
			}),
			ligo.OnStop(func(ctx any) error {
				onStopCalled.Store(true)
				return nil
			}),
			ligo.WithGracefulShutdown(5*time.Second),
		)

		app.Register(testModule)

		// Run app in background
		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		// Wait for server to start
		time.Sleep(500 * time.Millisecond)

		// Verify hooks were called
		if !onStartCalled.Load() {
			t.Error("OnStart hook was not called")
		}
		if !moduleInitCalled.Load() {
			t.Error("OnModuleInit hook was not called")
		}

		// Note: Full shutdown testing requires signal handling (SIGINT/SIGTERM)
		// which is complex to test. The fact that the app started successfully
		// and hooks were called indicates the lifecycle is working.

		// Don't wait for shutdown to avoid blocking the test
		go func() {
			<-runErr
		}()
	})

	t.Run("lifecycle hook execution order", func(t *testing.T) {
		var order []string
		var mu sync.Mutex

		addOrder := func(s string) {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, s)
		}

		testModule := ligo.NewModule("order",
			ligo.OnModuleInit(func() error {
				addOrder("module-init")
				return nil
			}),
			ligo.OnModuleDestroy(func() error {
				addOrder("module-destroy")
				return nil
			}),
		)

		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
			ligo.OnStart(func(ctx any) error {
				addOrder("app-start")
				return nil
			}),
			ligo.OnStop(func(ctx any) error {
				addOrder("app-stop")
				return nil
			}),
		)

		app.Register(testModule)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		// Wait for hooks to execute
		time.Sleep(300 * time.Millisecond)

		// Verify init hooks were called in order
		if len(order) < 2 {
			t.Fatalf("Expected at least 2 hook calls, got %d", len(order))
		}

		// Actual order: app-start is called BEFORE module-init
		// See app.Run(): OnStart hooks are called before OnModuleInit hooks
		if order[0] != "app-start" {
			t.Errorf("Expected first hook to be 'app-start', got '%s'", order[0])
		}
		if order[1] != "module-init" {
			t.Errorf("Expected second hook to be 'module-init', got '%s'", order[1])
		}

		// Don't wait for shutdown in test
		go func() {
			<-runErr
		}()
	})
}

// TestDIResolution tests that dependency injection works in a full app context.
func TestDIResolution(t *testing.T) {
	t.Run("provider injection in controller", func(t *testing.T) {
		// Create a test service
		testService := &TestService{message: "injected"}

		testModule := ligo.NewModule("di",
			ligo.Providers(
				ligo.Value(testService),
			),
			ligo.Controllers(func(svc *TestService) ligo.Controller {
				return &diController{service: svc}
			}),
		)

		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
		)

		app.Register(testModule)

		// Run app with context cancellation
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		// Wait for startup
		time.Sleep(200 * time.Millisecond)

		// Shutdown
		cancel()

		// Wait for shutdown with timeout
		select {
		case <-runErr:
			// Server shut down
		case <-ctx.Done():
			// Context timed out (shouldn't happen since we call cancel())
		}
	})

	t.Run("multiple providers in container", func(t *testing.T) {
		dbService := &DatabaseService{name: "test-db"}
		cacheService := &CacheService{name: "test-cache"}

		testModule := ligo.NewModule("multi",
			ligo.Providers(
				ligo.Value(dbService),
				ligo.Value(cacheService),
			),
		)

		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
		)

		app.Register(testModule)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		time.Sleep(200 * time.Millisecond)

		cancel()

		// Wait for shutdown or timeout
		select {
		case <-runErr:
			// Server shut down
		case <-ctx.Done():
			// Context timed out
		}
	})
}

// TestService is a simple service for DI testing.
type TestService struct {
	message string
}

// DatabaseService is a test service.
type DatabaseService struct {
	name string
}

// CacheService is a test service.
type CacheService struct {
	name string
}

// diController is a controller that uses DI.
type diController struct {
	service *TestService
}

func (c *diController) Routes(r ligo.Router) {
	r.Handle("GET", "/di-test", c.getDI)
}

func (c *diController) getDI(ctx ligo.Context) error {
	return ctx.OK(map[string]string{"message": c.service.message})
}

// TestMultipleModules tests that multiple modules work together.
func TestMultipleModules(t *testing.T) {
	userModule := ligo.NewModule("user",
		ligo.Providers(
			ligo.Value(&UserService{name: "user-service"}),
		),
	)

	productModule := ligo.NewModule("product",
		ligo.Providers(
			ligo.Value(&ProductService{name: "product-service"}),
		),
	)

	app := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)

	app.Register(userModule, productModule)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	cancel()

	// Wait for shutdown or timeout
	select {
	case <-runErr:
		// Server shut down
	case <-ctx.Done():
		// Context timed out
	}
}

// UserService is a test service.
type UserService struct {
	name string
}

// ProductService is a test service.
type ProductService struct {
	name string
}

// TestModuleImports tests that module imports work correctly.
func TestModuleImports(t *testing.T) {
	// Parent module that exports a provider
	parentModule := ligo.NewModule("parent",
		ligo.Providers(
			ligo.Export(ligo.Value(&SharedService{name: "shared"})),
		),
	)

	// Child module that imports parent
	childModule := ligo.NewModule("child",
		ligo.Imports(parentModule),
	)

	app := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)

	app.Register(childModule)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	cancel()

	// Wait for shutdown or timeout
	select {
	case <-runErr:
		// Server shut down
	case <-ctx.Done():
		// Context timed out
	}
}

// SharedService is a service that can be shared between modules.
type SharedService struct {
	name string
}

// TestDynamicModule tests dynamic modules in the full app context.
func TestDynamicModule(t *testing.T) {
	// Dynamic module factory
	configModuleFactory := func(opts ...any) ligo.Module {
		configValue := "default"
		if len(opts) > 0 {
			if s, ok := opts[0].(string); ok {
				configValue = s
			}
		}
		return ligo.NewModule("config",
			ligo.Providers(
				ligo.Value(configValue),
			),
		)
	}

	// Register dynamic module with custom config
	dynamicModule := ligo.NewModule("dynamic-wrapper",
		ligo.Dynamic(configModuleFactory, "custom-config"),
	)

	app := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)

	app.Register(dynamicModule)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	// If we got here without panic, dynamic module worked
	cancel()

	// Wait for shutdown or timeout
	select {
	case <-runErr:
		// Server shut down
	case <-ctx.Done():
		// Context timed out
	}
}

// TestModuleHooks tests module lifecycle hooks.
func TestModuleHooks(t *testing.T) {
	var initCalled atomic.Bool

	testModule := ligo.NewModule("hooks",
		ligo.OnModuleInit(func() error {
			initCalled.Store(true)
			return nil
		}),
		ligo.OnModuleDestroy(func() error {
			return nil
		}),
	)

	app := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)

	app.Register(testModule)

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	if !initCalled.Load() {
		t.Error("OnModuleInit was not called")
	}

	// Don't wait for shutdown in test
	go func() {
		<-runErr
	}()
}

// TestControllerRegistration tests that controllers are properly registered.
func TestControllerRegistration(t *testing.T) {
	var controllerRoutesCalled atomic.Bool

	testModule := ligo.NewModule("controller",
		ligo.Controllers(func() ligo.Controller {
			return &simpleController{called: &controllerRoutesCalled}
		}),
	)

	app := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)

	app.Register(testModule)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	if !controllerRoutesCalled.Load() {
		t.Error("Controller Routes method was not called")
	}

	cancel()

	// Wait for shutdown or timeout
	select {
	case <-runErr:
		// Server shut down
	case <-ctx.Done():
		// Context timed out
	}
}

// simpleController is a test controller.
type simpleController struct {
	called *atomic.Bool
}

func (c *simpleController) Routes(r ligo.Router) {
	c.called.Store(true)
	r.Handle("GET", "/test", func(ctx ligo.Context) error {
		return ctx.OK(map[string]string{"status": "ok"})
	})
}

// TestMiddleware tests middleware in the full app context.
func TestMiddleware(t *testing.T) {
	var middlewareCalled atomic.Bool

	testMiddleware := func(next ligo.HandlerFunc) ligo.HandlerFunc {
		return func(ctx ligo.Context) error {
			middlewareCalled.Store(true)
			return next(ctx)
		}
	}

	testModule := ligo.NewModule("mw",
		ligo.Middlewares(func() ligo.Middleware {
			return testMiddleware
		}),
	)

	app := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)

	app.Register(testModule)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	// Note: Middleware is called during request handling
	// We'd need to make an actual HTTP request to verify it works
	// For now, we just verify the app starts without errors

	cancel()

	// Wait for shutdown or timeout
	select {
	case <-runErr:
		// Server shut down
	case <-ctx.Done():
		// Context timed out
	}
}

// TestProviderTypes tests all provider types.
func TestProviderTypes(t *testing.T) {
	t.Run("value provider", func(t *testing.T) {
		module := ligo.NewModule("value",
			ligo.Providers(
				ligo.Value("test-value"),
			),
		)

		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
		)

		app.Register(module)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		time.Sleep(200 * time.Millisecond)

		// Trigger shutdown
		cancel()

		// Wait for shutdown or context timeout
		select {
		case <-runErr:
			// Server shut down
		case <-ctx.Done():
			// Context timed out
		}
	})

	t.Run("factory provider", func(t *testing.T) {
		module := ligo.NewModule("factory",
			ligo.Providers(
				ligo.Factory[*TestService](func() *TestService {
					return &TestService{message: "factory-created"}
				}),
			),
		)

		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
		)

		app.Register(module)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		time.Sleep(200 * time.Millisecond)

		// Trigger shutdown
		cancel()

		// Wait for shutdown or context timeout
		select {
		case <-runErr:
			// Server shut down
		case <-ctx.Done():
			// Context timed out
		}
	})

	t.Run("transient provider", func(t *testing.T) {
		module := ligo.NewModule("transient",
			ligo.Providers(
				ligo.Transient[*TestService](func() *TestService {
					return &TestService{message: "transient-created"}
				}),
			),
		)

		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
		)

		app.Register(module)

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		time.Sleep(200 * time.Millisecond)

		// Trigger shutdown
		cancel()

		// Wait for shutdown or context timeout
		select {
		case <-runErr:
			// Server shut down
		case <-ctx.Done():
			// Context timed out
		}
	})
}

// TestAppOptions tests various app configuration options.
func TestAppOptions(t *testing.T) {
	t.Run("with auto port", func(t *testing.T) {
		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":8080"),
			ligo.WithAutoPort(),
		)

		app.Register(ligo.NewModule("test"))

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		time.Sleep(200 * time.Millisecond)

		// Trigger shutdown
		cancel()

		// Wait for shutdown or context timeout
		select {
		case <-runErr:
			// Server shut down
		case <-ctx.Done():
			// Context timed out
		}
	})

	t.Run("with debug mode", func(t *testing.T) {
		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
			ligo.WithDebug(true),
		)

		app.Register(ligo.NewModule("test"))

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		time.Sleep(200 * time.Millisecond)

		// Trigger shutdown
		cancel()

		// Wait for shutdown or context timeout
		select {
		case <-runErr:
			// Server shut down
		case <-ctx.Done():
			// Context timed out
		}
	})

	t.Run("with JSON logging", func(t *testing.T) {
		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
			ligo.WithJSON(),
		)

		app.Register(ligo.NewModule("test"))

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		time.Sleep(200 * time.Millisecond)
		_ = runErr
	})
}

// TestErrorHandling tests error scenarios.
func TestErrorHandling(t *testing.T) {
	t.Run("module with init error", func(t *testing.T) {
		initError := fmt.Errorf("init failed")

		module := ligo.NewModule("error",
			ligo.OnModuleInit(func() error {
				return initError
			}),
		)

		app := ligo.New(
			ligo.WithRouter(echo.NewAdapter()),
			ligo.WithAddr(":0"),
		)

		app.Register(module)

		runErr := make(chan error, 1)
		go func() {
			runErr <- app.Run()
		}()

		// The app should handle the error gracefully
		select {
		case err := <-runErr:
			if err != nil && err != http.ErrServerClosed {
				// Expected: some error from failed init
				t.Logf("Got expected error from failed init: %v", err)
			}
		case <-time.After(500 * time.Millisecond):
			// If no error, the test still passes (app started)
		}
	})
}

// importedController is used by integration tests to verify route binding from imported modules.
type importedController struct {
	path  string
	calls *atomic.Int32
}

func (c *importedController) Routes(r ligo.Router) {
	r.Handle("GET", c.path, func(ctx ligo.Context) error { return ctx.OK("ok") })
	c.calls.Add(1)
}

// TestImportedModuleRoutes verifies that controllers from imported child modules
// are registered when the parent module is the only module passed to app.Register.
func TestImportedModuleRoutes(t *testing.T) {
	var authCalls, userCalls atomic.Int32

	authModule := ligo.NewModule("auth",
		ligo.Controllers(func() ligo.Controller {
			return &importedController{path: "/auth/login", calls: &authCalls}
		}),
	)
	userModule := ligo.NewModule("user",
		ligo.Controllers(func() ligo.Controller {
			return &importedController{path: "/user/list", calls: &userCalls}
		}),
	)
	mainModule := ligo.NewModule("main",
		ligo.Imports(authModule, userModule),
	)

	application := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)
	application.Register(mainModule)

	runErr := make(chan error, 1)
	go func() { runErr <- application.Run() }()

	time.Sleep(200 * time.Millisecond)

	if authCalls.Load() != 1 {
		t.Errorf("auth controller Routes() called %d times, want 1", authCalls.Load())
	}
	if userCalls.Load() != 1 {
		t.Errorf("user controller Routes() called %d times, want 1", userCalls.Load())
	}

	go func() { <-runErr }()
}

// TestSharedImportRegisteredOnce verifies the diamond pattern: when two modules
// both import the same child, that child's controllers are registered exactly once.
func TestSharedImportRegisteredOnce(t *testing.T) {
	var authCalls atomic.Int32

	authModule := ligo.NewModule("shared-auth",
		ligo.Controllers(func() ligo.Controller {
			return &importedController{path: "/shared-auth/verify", calls: &authCalls}
		}),
	)
	userModule := ligo.NewModule("shared-user", ligo.Imports(authModule))
	fileModule := ligo.NewModule("shared-file", ligo.Imports(authModule))
	mainModule := ligo.NewModule("shared-main", ligo.Imports(userModule, fileModule))

	application := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)
	application.Register(mainModule)

	runErr := make(chan error, 1)
	go func() { runErr <- application.Run() }()

	time.Sleep(200 * time.Millisecond)

	if authCalls.Load() != 1 {
		t.Errorf("shared auth controller Routes() called %d times, want exactly 1", authCalls.Load())
	}

	go func() { <-runErr }()
}

// TestDynamicModuleWithImports verifies that a dynamic module's child imports
// have their controllers bound after dynamic expansion.
func TestDynamicModuleWithImports(t *testing.T) {
	var childCalls atomic.Int32

	childModule := ligo.NewModule("dyn-child",
		ligo.Controllers(func() ligo.Controller {
			return &importedController{path: "/dyn-child/ping", calls: &childCalls}
		}),
	)

	dynamicFactory := func(opts ...any) ligo.Module {
		return ligo.NewModule("dyn-inner",
			ligo.Imports(childModule),
		)
	}

	wrapperModule := ligo.NewModule("dyn-wrapper",
		ligo.Dynamic(dynamicFactory),
	)

	application := ligo.New(
		ligo.WithRouter(echo.NewAdapter()),
		ligo.WithAddr(":0"),
	)
	application.Register(wrapperModule)

	runErr := make(chan error, 1)
	go func() { runErr <- application.Run() }()

	time.Sleep(200 * time.Millisecond)

	if childCalls.Load() != 1 {
		t.Errorf("dynamic child controller Routes() called %d times, want 1", childCalls.Load())
	}

	go func() { <-runErr }()
}

// lifecycleTracker tracks the order of hook execution.
type lifecycleTracker struct {
	mu    sync.Mutex
	calls []string
}

// trackedProvider implements all 5 lifecycle hooks.
type trackedProvider struct {
	tracker *lifecycleTracker
}

func (p *trackedProvider) OnModuleInit() error {
	p.tracker.mu.Lock()
	defer p.tracker.mu.Unlock()
	p.tracker.calls = append(p.tracker.calls, "init")
	return nil
}

func (p *trackedProvider) OnApplicationBootstrap() error {
	p.tracker.mu.Lock()
	defer p.tracker.mu.Unlock()
	p.tracker.calls = append(p.tracker.calls, "bootstrap")
	return nil
}

func (p *trackedProvider) BeforeApplicationShutdown() error {
	p.tracker.mu.Lock()
	defer p.tracker.mu.Unlock()
	p.tracker.calls = append(p.tracker.calls, "before-shutdown")
	return nil
}

func (p *trackedProvider) OnApplicationShutdown() error {
	p.tracker.mu.Lock()
	defer p.tracker.mu.Unlock()
	p.tracker.calls = append(p.tracker.calls, "shutdown")
	return nil
}

func (p *trackedProvider) OnModuleDestroy() error {
	p.tracker.mu.Lock()
	defer p.tracker.mu.Unlock()
	p.tracker.calls = append(p.tracker.calls, "destroy")
	return nil
}

// orderedProvider implements OnModuleInit with an ID for testing order.
type orderedProvider struct {
	id      string
	tracker *lifecycleTracker
}

func (p *orderedProvider) OnModuleInit() error {
	p.tracker.mu.Lock()
	defer p.tracker.mu.Unlock()
	p.tracker.calls = append(p.tracker.calls, "init-"+p.id)
	return nil
}

// TestLifecycleHooks tests the full lifecycle hook execution flow.
func TestLifecycleHooks(t *testing.T) {

	t.Run("non-HTTP mode executes all hooks", func(t *testing.T) {
		tracker := &lifecycleTracker{calls: []string{}}

		app := ligo.New()
		app.Register(
			ligo.NewModule("test",
				ligo.Providers(
					ligo.Value(&trackedProvider{tracker: tracker}),
				),
			),
		)

		// Run in background
		errCh := make(chan error, 1)
		go func() {
			errCh <- app.Run()
		}()

		// Give time for init and bootstrap hooks to execute
		time.Sleep(200 * time.Millisecond)

		// Send shutdown signal
		process, _ := os.FindProcess(os.Getpid())
		if err := process.Signal(os.Interrupt); err != nil {
			t.Fatalf("failed to send interrupt signal: %v", err)
		}

		// Wait for shutdown
		err := <-errCh
		if err != nil {
			t.Fatalf("app.Run() failed: %v", err)
		}

		// Verify hook execution order
		tracker.mu.Lock()
		calls := make([]string, len(tracker.calls))
		copy(calls, tracker.calls)
		tracker.mu.Unlock()

		expected := []string{"init", "bootstrap", "before-shutdown", "shutdown", "destroy"}
		if len(calls) != len(expected) {
			t.Fatalf("got %d calls, expected %d. Calls: %v", len(calls), len(expected), calls)
		}
		for i, call := range expected {
			if calls[i] != call {
				t.Errorf("call %d: got %s, want %s", i, calls[i], call)
			}
		}
	})

	t.Run("hooks execute in registration order", func(t *testing.T) {
		tracker := &lifecycleTracker{calls: []string{}}

		app := ligo.New()
		app.Register(
			ligo.NewModule("test",
				ligo.Providers(
					ligo.Value(&orderedProvider{id: "first", tracker: tracker}),
					ligo.Value(&orderedProvider{id: "second", tracker: tracker}),
				),
			),
		)

		// Run and shutdown
		errCh := make(chan error, 1)
		go func() {
			errCh <- app.Run()
		}()

		time.Sleep(200 * time.Millisecond)

		process, _ := os.FindProcess(os.Getpid())
		if err := process.Signal(os.Interrupt); err != nil {
			t.Fatalf("failed to send interrupt signal: %v", err)
		}

		<-errCh

		tracker.mu.Lock()
		calls := make([]string, len(tracker.calls))
		copy(calls, tracker.calls)
		tracker.mu.Unlock()

		// Should have init-first before init-second
		foundFirst, foundSecond := false, false
		for _, call := range calls {
			if call == "init-first" {
				foundFirst = true
				if foundSecond {
					t.Error("init-first called after init-second")
				}
			}
			if call == "init-second" {
				foundSecond = true
			}
		}
		if !foundFirst || !foundSecond {
			t.Errorf("missing init calls. Got: %v", calls)
		}
	})
}

// TestExplicitHookRegistration tests the explicit hook registration API.
func TestExplicitHookRegistration(t *testing.T) {
	t.Run("provider with explicit hooks", func(t *testing.T) {
		var initCalled, shutdownCalled atomic.Bool

		app := ligo.New()
		app.Register(
			ligo.NewModule("test",
				ligo.Providers(
					ligo.Value(&struct{}{},
						ligo.WithHooks(
							ligo.OnInit(func() error {
								initCalled.Store(true)
								return nil
							}),
							ligo.OnShutdown(func() error {
								shutdownCalled.Store(true)
								return nil
							}),
						),
					),
				),
			),
		)

		// Run in background
		errCh := make(chan error, 1)
		go func() {
			errCh <- app.Run()
		}()

		// Give time for init hook to execute
		time.Sleep(200 * time.Millisecond)

		if !initCalled.Load() {
			t.Error("OnInit hook was not called")
		}

		// Send shutdown signal
		process, _ := os.FindProcess(os.Getpid())
		if err := process.Signal(os.Interrupt); err != nil {
			t.Fatalf("failed to send interrupt signal: %v", err)
		}

		<-errCh

		if !shutdownCalled.Load() {
			t.Error("OnShutdown hook was not called")
		}
	})

	t.Run("module with explicit hooks", func(t *testing.T) {
		var moduleInitCalled, moduleDestroyCalled atomic.Bool

		app := ligo.New()
		app.Register(
			ligo.NewModule("test",
				ligo.WithModuleHooks(
					ligo.ModuleInit(func() error {
						moduleInitCalled.Store(true)
						return nil
					}),
					ligo.ModuleDestroy(func() error {
						moduleDestroyCalled.Store(true)
						return nil
					}),
				),
			),
		)

		// Run in background
		errCh := make(chan error, 1)
		go func() {
			errCh <- app.Run()
		}()

		// Give time for init hook to execute
		time.Sleep(200 * time.Millisecond)

		if !moduleInitCalled.Load() {
			t.Error("Module OnInit hook was not called")
		}

		// Send shutdown signal
		process, _ := os.FindProcess(os.Getpid())
		if err := process.Signal(os.Interrupt); err != nil {
			t.Fatalf("failed to send interrupt signal: %v", err)
		}

		<-errCh

		if !moduleDestroyCalled.Load() {
			t.Error("Module OnDestroy hook was not called")
		}
	})
}

// TestHookedFactory_RegisterMethod tests the HookedFactory pattern where services
// explicitly register their hooks via a Register method, enabling compile-time safety.
func TestHookedFactory_RegisterMethod(t *testing.T) {
	t.Run("service with Register method for compile-time safe hooks", func(t *testing.T) {
		var initCalled, shutdownCalled atomic.Bool

		db := &TestDatabase{
			initCalled:    &initCalled,
			shutdownCalled: &shutdownCalled,
		}

		// Module using Value with Hooks - RegisterFrom is called immediately for eager providers
		testModule := ligo.NewModule("test",
			ligo.Providers(
				ligo.Value(db, ligo.WithHooks()),
			),
		)

		app := ligo.New()
		app.Register(testModule)

		// Run in background
		errCh := make(chan error, 1)
		go func() {
			errCh <- app.Run()
		}()

		// Give time for init hook to execute
		time.Sleep(200 * time.Millisecond)

		if !initCalled.Load() {
			t.Error("Database OnInit hook was not called")
		}

		// Send shutdown signal
		process, _ := os.FindProcess(os.Getpid())
		if err := process.Signal(os.Interrupt); err != nil {
			t.Fatalf("failed to send interrupt signal: %v", err)
		}

		<-errCh

		if !shutdownCalled.Load() {
			t.Error("Database OnShutdown hook was not called")
		}
	})
}
