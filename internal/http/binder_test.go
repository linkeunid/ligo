package http

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/linkeunid/ligo/internal/di"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
)

type mockRouter struct {
	routes []string
}

func (m *mockRouter) Group(prefix string) Router { return m }
func (m *mockRouter) Use(mw ...Middleware)        {}
func (m *mockRouter) Serve(addr string) error     { return nil }
func (m *mockRouter) Handle(method, path string, _ HandlerFunc) {
	m.routes = append(m.routes, method+":"+path)
}

type mockController struct {
	path  string
	calls *atomic.Int32
}

func (c *mockController) Routes(r Router) {
	r.Handle("GET", c.path, func(ctx Context) error { return nil })
	c.calls.Add(1)
}

func newMockController(path string, calls *atomic.Int32) func() Controller {
	return func() Controller {
		return &mockController{path: path, calls: calls}
	}
}

func TestBindControllers_ImportRecursion(t *testing.T) {
	t.Run("controllers from imported child module are bound", func(t *testing.T) {
		var authCalls, userCalls atomic.Int32

		authMod := module.New("auth",
			module.Controllers(newMockController("/auth/login", &authCalls)),
		)
		mainMod := module.New("main",
			module.Imports(authMod),
			module.Controllers(newMockController("/health", &userCalls)),
		)

		router := &mockRouter{}
		c := di.New()
		log := logger.New()
		binder := NewBinder(c, router, log)

		_, err := binder.BindControllers([]module.Module{mainMod})
		if err != nil {
			t.Fatalf("BindControllers() error = %v", err)
		}

		if authCalls.Load() != 1 {
			t.Errorf("auth controller Routes() called %d times, want 1", authCalls.Load())
		}
		if userCalls.Load() != 1 {
			t.Errorf("main controller Routes() called %d times, want 1", userCalls.Load())
		}
	})

	t.Run("deeply nested imports are all bound", func(t *testing.T) {
		var dbCalls, repoCalls, svcCalls atomic.Int32

		dbMod := module.New("db",
			module.Controllers(newMockController("/db/ping", &dbCalls)),
		)
		repoMod := module.New("repo",
			module.Imports(dbMod),
			module.Controllers(newMockController("/repo/list", &repoCalls)),
		)
		svcMod := module.New("svc",
			module.Imports(repoMod),
			module.Controllers(newMockController("/svc/call", &svcCalls)),
		)

		router := &mockRouter{}
		c := di.New()
		log := logger.New()
		binder := NewBinder(c, router, log)

		_, err := binder.BindControllers([]module.Module{svcMod})
		if err != nil {
			t.Fatalf("BindControllers() error = %v", err)
		}

		if dbCalls.Load() != 1 {
			t.Errorf("db controller Routes() called %d times, want 1", dbCalls.Load())
		}
		if repoCalls.Load() != 1 {
			t.Errorf("repo controller Routes() called %d times, want 1", repoCalls.Load())
		}
		if svcCalls.Load() != 1 {
			t.Errorf("svc controller Routes() called %d times, want 1", svcCalls.Load())
		}
	})

	t.Run("module with no imports binds only own controllers", func(t *testing.T) {
		var calls atomic.Int32

		mod := module.New("standalone",
			module.Controllers(newMockController("/ping", &calls)),
		)

		router := &mockRouter{}
		c := di.New()
		log := logger.New()
		binder := NewBinder(c, router, log)

		_, err := binder.BindControllers([]module.Module{mod})
		if err != nil {
			t.Fatalf("BindControllers() error = %v", err)
		}

		if calls.Load() != 1 {
			t.Errorf("controller Routes() called %d times, want 1", calls.Load())
		}
	})
}

type myBinderService struct{}

func TestBindController_MissingDep_ReturnsErrControllerBinding(t *testing.T) {
	c := di.New()
	// Do NOT register myBinderService — binder should return ErrControllerBinding

	mod := module.New("user",
		module.Controllers(func(svc *myBinderService) Controller {
			return nil
		}),
	)

	router := &mockRouter{}
	log := logger.New()
	binder := NewBinder(c, router, log)

	_, err := binder.BindControllers([]module.Module{mod})
	if err == nil {
		t.Fatal("expected error")
	}

	var bindErr *ErrControllerBinding
	if !errors.As(err, &bindErr) {
		t.Fatalf("expected *ErrControllerBinding, got %T: %v", err, err)
	}
	if bindErr.Module != "user" {
		t.Errorf("Module = %q, want %q", bindErr.Module, "user")
	}
}
