package lifecycle

import (
	"errors"
	"testing"
)

func TestHookRegistry_AllSetters(t *testing.T) {
	r := NewHookRegistry()
	want := errors.New("ok")
	hook := func() error { return want }

	r.OnInit(hook).OnBootstrap(hook).BeforeShutdown(hook).OnShutdown(hook).OnDestroy(hook)

	h := r.ToHooks()
	for name, fn := range map[string]func() error{
		"OnInit":           h.OnInit,
		"OnBootstrap":      h.OnBootstrap,
		"OnBeforeShutdown": h.OnBeforeShutdown,
		"OnShutdown":       h.OnShutdown,
		"OnDestroy":        h.OnDestroy,
	} {
		if fn == nil {
			t.Errorf("%s: nil", name)
			continue
		}
		if err := fn(); !errors.Is(err, want) {
			t.Errorf("%s err = %v, want %v", name, err, want)
		}
	}
}

// registerableService implements Registerable to verify RegisterFrom dispatch.
type registerableService struct {
	initCalled bool
}

func (s *registerableService) Init() error { s.initCalled = true; return nil }

func (s *registerableService) Register(r *HookRegistry) {
	r.OnInit(s.Init)
}

func TestRegisterFrom_DispatchesToRegisterable(t *testing.T) {
	r := NewHookRegistry()
	svc := &registerableService{}
	r.RegisterFrom(svc)

	h := r.ToHooks()
	if h.OnInit == nil {
		t.Fatal("OnInit not registered")
	}
	if err := h.OnInit(); err != nil {
		t.Fatal(err)
	}
	if !svc.initCalled {
		t.Error("Init not invoked")
	}
}

func TestRegisterFrom_NonRegisterableIsNoop(t *testing.T) {
	r := NewHookRegistry()
	r.RegisterFrom("not a registerable")
	if r.ToHooks().OnInit != nil {
		t.Error("non-registerable should not populate hooks")
	}
}

func TestModuleHookRegistry_AppendsAcrossCalls(t *testing.T) {
	r := NewModuleHookRegistry()
	r.OnInit(func() error { return nil })
	r.OnInit(func() error { return nil })
	r.OnDestroy(func() error { return nil })

	if got := len(r.GetInitHooks()); got != 2 {
		t.Errorf("init hooks = %d, want 2", got)
	}
	if got := len(r.GetDestroyHooks()); got != 1 {
		t.Errorf("destroy hooks = %d, want 1", got)
	}
}
