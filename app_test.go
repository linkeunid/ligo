package ligo

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/linkeunid/ligo/internal/app"
	"github.com/linkeunid/ligo/internal/core/lifecycle"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/di"
)

type testSvc struct {
	name string
}

// waitForCondition polls cond every millisecond up to timeout. Returns true
// when cond becomes true, false on timeout. Used to replace fixed-duration
// time.Sleep waits in lifecycle tests — tightens the wall-clock cost from
// the blanket sleep duration to the actual time the condition needs.
func waitForCondition(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return cond()
}

// signalShutdown sends SIGINT to the current process and waits for errCh.
func signalShutdown(t *testing.T, errCh chan error) {
	t.Helper()
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)
	<-errCh
}

func TestAppNew(t *testing.T) {
	app := New()
	if app == nil {
		t.Fatal("expected non-nil app")
	}
	if app.started {
		t.Fatal("expected app to not be started")
	}
}

func TestAppRegister(t *testing.T) {
	app := New()
	mod := NewModule("test")
	app.Register(mod)
	if len(app.modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(app.modules))
	}
}

func TestAppProvide(t *testing.T) {
	app := New()
	app.Provide(Value(&testSvc{name: "svc"}))
	if len(app.providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(app.providers))
	}
}

func TestAppRunResolvesModules(t *testing.T) {
	app := New()
	app.Register(NewModule(
		"test",
		Providers(
			Value(&testSvc{name: "svc"}),
		),
	))

	// Run app in background since it will wait for shutdown signal
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	if !waitForCondition(2*time.Second, func() bool { return app.container.Load() != nil }) {
		t.Fatal("container not built within 2s")
	}

	svc := di.MustResolve[*testSvc](app.Container())
	if svc.name != "svc" {
		t.Fatalf("expected 'svc', got %s", svc.name)
	}

	signalShutdown(t, errCh)
}

func TestAppRunLocksApp(t *testing.T) {
	app := New()
	app.Register(NewModule("test"))

	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	// Use container as start signal — atomic.Pointer is safe to read
	// concurrently. The plain bool app.started would race with Run.
	if !waitForCondition(2*time.Second, func() bool { return app.container.Load() != nil }) {
		t.Fatal("app did not start within 2s")
	}

	defer func() {
		signalShutdown(t, errCh)
		if r := recover(); r == nil {
			t.Fatal("expected panic on Register after Run")
		}
	}()

	app.Register(NewModule("another"))
}

func TestAppProvideLocksApp(t *testing.T) {
	app := New()

	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	// Use container as start signal — atomic.Pointer is safe to read
	// concurrently. The plain bool app.started would race with Run.
	if !waitForCondition(2*time.Second, func() bool { return app.container.Load() != nil }) {
		t.Fatal("app did not start within 2s")
	}

	defer func() {
		signalShutdown(t, errCh)
		if r := recover(); r == nil {
			t.Fatal("expected panic on Provide after Run")
		}
	}()

	app.Provide(Value(&testSvc{}))
}

func TestAppContainerEscapeHatch(t *testing.T) {
	app := New()
	app.Register(NewModule(
		"test",
		Providers(Value(&testSvc{name: "hatch"})),
	))

	errCh := make(chan error, 1)
	go func() { errCh <- app.Run() }()

	if !waitForCondition(2*time.Second, func() bool { return app.container.Load() != nil }) {
		signalShutdown(t, errCh)
		t.Fatal("expected container escape hatch within 2s")
	}
	c := app.Container()

	svc := di.MustResolve[*testSvc](c)
	if svc.name != "hatch" {
		signalShutdown(t, errCh)
		t.Fatalf("expected 'hatch', got %s", svc.name)
	}
	signalShutdown(t, errCh)
}

func TestAppContainerPanicsBeforeRun(t *testing.T) {
	app := New()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic before Run()")
		}
	}()

	_ = app.Container()
}

func TestExecuteHooksParallel_JoinsErrors(t *testing.T) {
	tasks := []hookTask{
		{hook: func() error { return errors.New("first") }},
		{hook: func() error { return nil }},
		{hook: func() error { return errors.New("third") }},
	}
	err := executeHooksParallel(tasks)
	if err == nil {
		t.Fatal("expected joined error")
	}
	if !strings.Contains(err.Error(), "first") {
		t.Errorf("missing 'first': %v", err)
	}
	if !strings.Contains(err.Error(), "third") {
		t.Errorf("missing 'third': %v", err)
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("expected errors.Join result, got %T", err)
	}
	if got := len(joined.Unwrap()); got != 2 {
		t.Errorf("expected 2 wrapped errors, got %d", got)
	}
}

func TestExecuteHooksParallel_NoErrorReturnsNil(t *testing.T) {
	tasks := []hookTask{{hook: func() error { return nil }}}
	if err := executeHooksParallel(tasks); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestExecuteHooksParallel_EmptyReturnsNil(t *testing.T) {
	if err := executeHooksParallel(nil); err != nil {
		t.Errorf("expected nil for empty tasks, got %v", err)
	}
}

func TestAppShutdown_ReturnsJoinedErrors(t *testing.T) {
	a := New()
	a.opts.logger = logger.Noop()
	a.moduleHooks = &app.ModuleHooks{
		Providers: []lifecycle.Hooks{
			{OnDestroy: func() error { return errors.New("p1-destroy") }},
			{OnShutdown: func() error { return errors.New("p2-shutdown") }},
			{OnBeforeShutdown: func() error { return errors.New("p3-before") }},
		},
	}

	err := a.shutdown()
	if err == nil {
		t.Fatal("expected joined shutdown error")
	}
	for _, want := range []string{"p1-destroy", "p2-shutdown", "p3-before"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in: %v", want, err)
		}
	}
	for _, prefix := range []string{"OnModuleDestroy", "OnApplicationShutdown", "BeforeApplicationShutdown"} {
		if !strings.Contains(err.Error(), prefix) {
			t.Errorf("missing prefix %q in: %v", prefix, err)
		}
	}
}

func TestAppShutdown_NoHooksReturnsNil(t *testing.T) {
	a := New()
	a.opts.logger = logger.Noop()
	a.moduleHooks = &app.ModuleHooks{}
	if err := a.shutdown(); err != nil {
		t.Errorf("expected nil shutdown error, got %v", err)
	}
}
