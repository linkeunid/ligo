package ligo

import (
	"testing"

	"github.com/linkeunid/ligo/internal/container"
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
	app.Register(NewModule("test",
		Providers(
			Value(&testSvc{name: "svc"}),
		),
	))

	app.Run()

	if app.container == nil {
		t.Fatal("expected container to be built after Run()")
	}

	svc := container.Resolve[*testSvc](app.container)
	if svc.name != "svc" {
		t.Fatalf("expected 'svc', got %s", svc.name)
	}
}

func TestAppRunLocksApp(t *testing.T) {
	app := New()
	app.Register(NewModule("test"))
	app.Run()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on Register after Run")
		}
	}()

	app.Register(NewModule("another"))
}

func TestAppProvideLocksApp(t *testing.T) {
	app := New()
	app.Run()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on Provide after Run")
		}
	}()

	app.Provide(Value(&testSvc{}))
}

func TestAppContainerEscapeHatch(t *testing.T) {
	app := New()
	app.Register(NewModule("test",
		Providers(Value(&testSvc{name: "hatch"})),
	))
	app.Run()

	c := app.Container()
	if c == nil {
		t.Fatal("expected container escape hatch")
	}

	svc := container.Resolve[*testSvc](c)
	if svc.name != "hatch" {
		t.Fatalf("expected 'hatch', got %s", svc.name)
	}
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
