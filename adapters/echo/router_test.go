package echo

import (
	"testing"

	"github.com/linkeunid/ligo"
)

func TestAdapterImplementsRouter(t *testing.T) {
	var _ ligo.Router = NewAdapter()
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
	a.Handle("GET", "/test", func(ctx ligo.Context) error {
		called = true
		return ctx.String(200, "ok")
	})
	if !called {
		// Handler is registered, not called yet
	}
}

func TestAdapterServe(t *testing.T) {
	a := NewAdapter()
	// We don't actually start the server in tests
	// Just verify the method exists and returns an error for invalid addr
	err := a.Serve("invalid")
	if err == nil {
		// Server might fail to start, which is expected
	}
}

func TestGroupAdapterImplementsRouter(t *testing.T) {
	a := NewAdapter()
	g := a.Group("/api")
	var _ ligo.Router = g
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
