package ligo

import (
	"os"
	"testing"
	"time"

	"github.com/linkeunid/ligo/internal/di"
)

type testSvc struct {
	name string
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

	// Wait for app to start and build container
	time.Sleep(100 * time.Millisecond)

	container := app.Container()
	if container == nil {
		t.Fatal("expected container to be built after Run()")
	}

	svc := di.MustResolve[*testSvc](container)
	if svc.name != "svc" {
		t.Fatalf("expected 'svc', got %s", svc.name)
	}

	// Send shutdown signal to stop the app
	process, _ := os.FindProcess(os.Getpid())
	_ = process.Signal(os.Interrupt)

	// Wait for app to stop
	<-errCh
}

func TestAppRunLocksApp(t *testing.T) {
	app := New()
	app.Register(NewModule("test"))

	// Run app in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	// Wait for app to start
	time.Sleep(100 * time.Millisecond)

	defer func() {
		// Send shutdown signal to stop the app
		process, _ := os.FindProcess(os.Getpid())
		_ = process.Signal(os.Interrupt)
		<-errCh

		if r := recover(); r == nil {
			t.Fatal("expected panic on Register after Run")
		}
	}()

	app.Register(NewModule("another"))
}

func TestAppProvideLocksApp(t *testing.T) {
	app := New()

	// Run app in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	// Wait for app to start
	time.Sleep(100 * time.Millisecond)

	defer func() {
		// Send shutdown signal to stop the app
		process, _ := os.FindProcess(os.Getpid())
		_ = process.Signal(os.Interrupt)
		<-errCh

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

	// Run app in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	// Wait for app to start
	time.Sleep(100 * time.Millisecond)

	c := app.Container()
	if c == nil {
		// Send shutdown signal before failing
		process, _ := os.FindProcess(os.Getpid())
		_ = process.Signal(os.Interrupt)
		<-errCh
		t.Fatal("expected container escape hatch")
	}

	svc := di.MustResolve[*testSvc](c)
	if svc.name != "hatch" {
		// Send shutdown signal before failing
		process, _ := os.FindProcess(os.Getpid())
		_ = process.Signal(os.Interrupt)
		<-errCh
		t.Fatalf("expected 'hatch', got %s", svc.name)
	}

	// Send shutdown signal to stop the app
	process, _ := os.FindProcess(os.Getpid())
	_ = process.Signal(os.Interrupt)
	<-errCh
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
