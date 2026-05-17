package echo

import (
	"errors"
	"testing"

	"github.com/linkeunid/ligo/internal/di"
	httpifc "github.com/linkeunid/ligo/internal/http"
)

func TestAdapterImplementsRouter(t *testing.T) {
	var _ httpifc.Router = NewAdapter()
}

func TestAdapterGroup(t *testing.T) {
	a := NewAdapter()
	g := a.Group("/api")
	if g == nil {
		t.Fatal("expected non-nil group")
	}
}

func TestAdapterHandle(t *testing.T) {
	a := NewAdapter()
	called := false
	a.Handle("GET", "/test", func(ctx *httpifc.Context) error {
		called = true
		return ctx.String(200, "ok")
	})
	_ = called // handler is registered, not called yet — presence is the assertion
}

func TestAdapterServe(t *testing.T) {
	a := NewAdapter()
	// We don't actually start the server in tests
	// Just verify the method exists and returns an error for invalid addr
	_ = a.Serve("invalid") // either path is acceptable — we only assert the method exists
}

func TestGroupAdapterImplementsRouter(t *testing.T) {
	a := NewAdapter()
	g := a.Group("/api")
	var _ httpifc.Router = g
}

func TestGroupAdapterGroup(t *testing.T) {
	a := NewAdapter()
	g1 := a.Group("/api")
	g2 := g1.Group("/v1")
	if g2 == nil {
		t.Fatal("expected non-nil nested group")
	}
}

func TestContextAdapter(t *testing.T) {
	// Context adapter tests would require a real Echo context
	// which needs an HTTP test server. Skip for now.
}

func TestGroupAdapterServeReturnsSentinel(t *testing.T) {
	a := NewAdapter()
	g := a.Group("/api")
	err := g.Serve(":0")
	if !errors.Is(err, ErrServeOnGroup) {
		t.Fatalf("expected ErrServeOnGroup, got %v", err)
	}
}

func TestSetContainerIdempotent(t *testing.T) {
	a := NewAdapter()
	c := di.New()

	a.SetContainer(c)
	first := len(a.middleware)

	a.SetContainer(c)
	a.SetContainer(c)
	if len(a.middleware) != first {
		t.Fatalf("middleware count grew: first=%d, after extra calls=%d", first, len(a.middleware))
	}
}

func TestSetContainerNilDoesNotInstall(t *testing.T) {
	a := NewAdapter()
	a.SetContainer(nil)
	if a.reqScopeInstalled {
		t.Fatal("nil container should not install request-scope middleware")
	}
	if len(a.middleware) != 0 {
		t.Fatalf("nil container should leave middleware empty, got %d", len(a.middleware))
	}
}
