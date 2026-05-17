package http

import (
	"errors"
	"testing"
)

func TestRouteBuilder_NilHandlerPanics(t *testing.T) {
	router := &mockRouter{}
	rb := NewRouteBuilder(router, "GET", "/x")

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		for _, want := range []string{"GET", "/x", ".Handle("} {
			if !contains(msg, want) {
				t.Errorf("panic missing %q: %q", want, msg)
			}
		}
	}()
	rb.Handle()
}

func TestRouteBuilder_GuardDeniedReturnsSentinel(t *testing.T) {
	router := &mockRouter{}
	var captured HandlerFunc
	router.handleFunc = func(_, _ string, h HandlerFunc) { captured = h }

	NewRouteBuilder(router, "GET", "/x").
		Guard(func(Context) (bool, error) { return false, nil }).
		Handle(func(Context) error { return nil })

	if captured == nil {
		t.Fatal("handler not registered")
	}
	err := captured(nil)
	if !errors.Is(err, ErrGuardDenied) {
		t.Errorf("expected ErrGuardDenied, got %v", err)
	}
}

func TestRouteBuilder_GuardAllowsThroughHandler(t *testing.T) {
	router := &mockRouter{}
	var captured HandlerFunc
	router.handleFunc = func(_, _ string, h HandlerFunc) { captured = h }
	called := false

	NewRouteBuilder(router, "GET", "/x").
		Guard(func(Context) (bool, error) { return true, nil }).
		Handle(func(Context) error { called = true; return nil })

	if captured == nil {
		t.Fatal("handler not registered")
	}
	if err := captured(nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not invoked")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
