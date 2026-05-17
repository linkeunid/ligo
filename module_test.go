package ligo

import (
	"testing"
)

func TestNewModule(t *testing.T) {
	mod := NewModule(
		"users",
		Providers(
			Factory[*testService](func() *testService { return &testService{} }),
		),
	)
	if mod.Name != "users" {
		t.Fatalf("expected name 'users', got %s", mod.Name)
	}
	if len(mod.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(mod.Providers))
	}
}

func TestModuleWithImports(t *testing.T) {
	child := NewModule("child")
	parent := NewModule(
		"parent",
		Imports(child),
	)
	if len(parent.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(parent.Imports))
	}
}

func TestModuleWithControllers(t *testing.T) {
	mod := NewModule(
		"users",
		Controllers(func() Controller { return nil }),
	)
	if len(mod.Controllers) != 1 {
		t.Fatalf("expected 1 controller, got %d", len(mod.Controllers))
	}
}
