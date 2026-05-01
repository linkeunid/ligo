package ligo

import (
	"testing"
)

type testService struct {
	name string
}

func TestNewModule(t *testing.T) {
	mod := NewModule("users",
		Providers(
			Factory[*testService](func() *testService { return &testService{} }),
		),
	)
	if mod.name != "users" {
		t.Fatalf("expected name 'users', got %s", mod.name)
	}
	if len(mod.providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(mod.providers))
	}
}

func TestModuleWithImports(t *testing.T) {
	child := NewModule("child")
	parent := NewModule("parent",
		Imports(child),
	)
	if len(parent.imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(parent.imports))
	}
}

func TestModuleWithControllers(t *testing.T) {
	mod := NewModule("users",
		Controllers(func() Controller { return nil }),
	)
	if len(mod.controllers) != 1 {
		t.Fatalf("expected 1 controller, got %d", len(mod.controllers))
	}
}
