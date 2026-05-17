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

func TestRouteBuilder_PipeRunsBeforeHandler(t *testing.T) {
	router := &mockRouter{}
	var captured HandlerFunc
	router.handleFunc = func(_, _ string, h HandlerFunc) { captured = h }

	order := []string{}
	NewRouteBuilder(router, "GET", "/x").
		Pipe(func(Context) error { order = append(order, "pipe"); return nil }).
		Handle(func(Context) error { order = append(order, "handler"); return nil })

	if err := captured(nil); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "pipe" || order[1] != "handler" {
		t.Errorf("order = %v", order)
	}
}

func TestRouteBuilder_PipeErrorShortCircuits(t *testing.T) {
	router := &mockRouter{}
	var captured HandlerFunc
	router.handleFunc = func(_, _ string, h HandlerFunc) { captured = h }

	want := errors.New("pipe boom")
	handlerCalled := false
	NewRouteBuilder(router, "GET", "/x").
		Pipe(func(Context) error { return want }).
		Handle(func(Context) error { handlerCalled = true; return nil })

	if err := captured(nil); !errors.Is(err, want) {
		t.Errorf("err = %v", err)
	}
	if handlerCalled {
		t.Error("handler should not run after pipe error")
	}
}

func TestRouteBuilder_InterceptorWrapsCycle(t *testing.T) {
	router := &mockRouter{}
	var captured HandlerFunc
	router.handleFunc = func(_, _ string, h HandlerFunc) { captured = h }

	order := []string{}
	NewRouteBuilder(router, "GET", "/x").
		Intercept(func(ctx Context, next HandlerFunc) error {
			order = append(order, "before")
			err := next(ctx)
			order = append(order, "after")
			return err
		}).
		Handle(func(Context) error { order = append(order, "handler"); return nil })

	if err := captured(nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"before", "handler", "after"}
	for i, s := range want {
		if order[i] != s {
			t.Errorf("step %d = %q, want %q", i, order[i], s)
		}
	}
}

func TestRouteBuilder_MiddlewareAppliesInReverseOrder(t *testing.T) {
	router := &mockRouter{}
	var captured HandlerFunc
	router.handleFunc = func(_, _ string, h HandlerFunc) { captured = h }

	order := []string{}
	mw := func(label string) Middleware {
		return func(next HandlerFunc) HandlerFunc {
			return func(c Context) error {
				order = append(order, label)
				return next(c)
			}
		}
	}

	NewRouteBuilder(router, "GET", "/x").
		Use(mw("outer"), mw("inner")).
		Handle(func(Context) error { return nil })

	if err := captured(nil); err != nil {
		t.Fatal(err)
	}
	if order[0] != "outer" || order[1] != "inner" {
		t.Errorf("middleware order = %v", order)
	}
}

func TestRouteBuilder_FilterTransformsError(t *testing.T) {
	router := &mockRouter{}
	var captured HandlerFunc
	router.handleFunc = func(_, _ string, h HandlerFunc) { captured = h }

	transformed := errors.New("transformed")
	NewRouteBuilder(router, "GET", "/x").
		Filter(func(_ error, _ Context) error { return transformed }).
		Handle(func(Context) error { return errors.New("original") })

	if err := captured(nil); !errors.Is(err, transformed) {
		t.Errorf("filter did not transform err: %v", err)
	}
}

func TestRouteBuilder_FilterNoErrorPassesThrough(t *testing.T) {
	router := &mockRouter{}
	var captured HandlerFunc
	router.handleFunc = func(_, _ string, h HandlerFunc) { captured = h }

	NewRouteBuilder(router, "GET", "/x").
		Filter(func(_ error, _ Context) error { return nil }).
		Handle(func(Context) error { return nil })

	if err := captured(nil); err != nil {
		t.Errorf("no error path returned err: %v", err)
	}
}
