package module

import (
	"testing"
)

func TestNew(t *testing.T) {
	m := New("test")
	if m.Name != "test" {
		t.Errorf("New() Name = %v, want %v", m.Name, "test")
	}
	if len(m.Providers) != 0 {
		t.Errorf("New() Providers = %v, want empty", m.Providers)
	}
	if len(m.Controllers) != 0 {
		t.Errorf("New() Controllers = %v, want empty", m.Controllers)
	}
	if len(m.Imports) != 0 {
		t.Errorf("New() Imports = %v, want empty", m.Imports)
	}
}

func TestProviders(t *testing.T) {
	provider1 := "provider1"
	provider2 := "provider2"

	opt := Providers(provider1, provider2)
	m := &Module{}
	opt(m)

	if len(m.Providers) != 2 {
		t.Errorf("Providers() added %d providers, want 2", len(m.Providers))
	}
	if m.Providers[0] != provider1 {
		t.Errorf("Providers()[0] = %v, want %v", m.Providers[0], provider1)
	}
	if m.Providers[1] != provider2 {
		t.Errorf("Providers()[1] = %v, want %v", m.Providers[1], provider2)
	}
}

func TestImports(t *testing.T) {
	module1 := New("module1")
	module2 := New("module2")

	opt := Imports(module1, module2)
	m := &Module{}
	opt(m)

	if len(m.Imports) != 2 {
		t.Errorf("Imports() added %d modules, want 2", len(m.Imports))
	}
	if m.Imports[0].Name != "module1" {
		t.Errorf("Imports()[0].Name = %v, want %v", m.Imports[0].Name, "module1")
	}
	if m.Imports[1].Name != "module2" {
		t.Errorf("Imports()[1].Name = %v, want %v", m.Imports[1].Name, "module2")
	}
}

func TestControllers(t *testing.T) {
	ctrl1 := func() string { return "ctrl1" }
	ctrl2 := func() int { return 42 }

	opt := Controllers(ctrl1, ctrl2)
	m := &Module{}
	opt(m)

	if len(m.Controllers) != 2 {
		t.Errorf("Controllers() added %d controllers, want 2", len(m.Controllers))
	}
	if m.Controllers[0].Fn == nil {
		t.Errorf("Controllers()[0].Fn is nil")
	}
	if m.Controllers[1].Fn == nil {
		t.Errorf("Controllers()[1].Fn is nil")
	}
}

func TestMiddlewares(t *testing.T) {
	mw1 := func() string { return "mw1" }
	mw2 := func() int { return 42 }

	opt := Middlewares(mw1, mw2)
	m := &Module{}
	opt(m)

	if len(m.Middlewares) != 2 {
		t.Errorf("Middlewares() added %d middlewares, want 2", len(m.Middlewares))
	}
	if m.Middlewares[0].Fn == nil {
		t.Errorf("Middlewares()[0].Fn is nil")
	}
	if m.Middlewares[1].Fn == nil {
		t.Errorf("Middlewares()[1].Fn is nil")
	}
}

func TestOnModuleInit(t *testing.T) {
	called := false
	initFn := func() error {
		called = true
		return nil
	}

	opt := OnModuleInit(initFn)
	m := &Module{}
	opt(m)

	if len(m.OnInit) != 1 {
		t.Errorf("OnModuleInit() added %d hooks, want 1", len(m.OnInit))
	}

	err := m.OnInit[0]()
	if err != nil {
		t.Errorf("OnModuleInit hook returned error: %v", err)
	}
	if !called {
		t.Error("OnModuleInit hook was not called")
	}
}

func TestOnModuleDestroy(t *testing.T) {
	called := false
	destroyFn := func() error {
		called = true
		return nil
	}

	opt := OnModuleDestroy(destroyFn)
	m := &Module{}
	opt(m)

	if len(m.OnDestroy) != 1 {
		t.Errorf("OnModuleDestroy() added %d hooks, want 1", len(m.OnDestroy))
	}

	err := m.OnDestroy[0]()
	if err != nil {
		t.Errorf("OnModuleDestroy hook returned error: %v", err)
	}
	if !called {
		t.Error("OnModuleDestroy hook was not called")
	}
}

func TestDynamic(t *testing.T) {
	factory := func(opts ...any) Module {
		name := "default"
		if len(opts) > 0 {
			if n, ok := opts[0].(string); ok {
				name = n
			}
		}
		return New(name)
	}

	opt := Dynamic(factory, "custom")
	m := &Module{}
	opt(m)

	if m.Dynamic == nil {
		t.Fatal("Dynamic() did not set Dynamic field")
	}
	if m.Dynamic.Factory == nil {
		t.Error("Dynamic() did not set Factory")
	}
	if len(m.Dynamic.Options) != 1 {
		t.Errorf("Dynamic() Options length = %d, want 1", len(m.Dynamic.Options))
	}

	resultMod := m.Dynamic.Factory("test")
	if resultMod.Name != "test" {
		t.Errorf("Dynamic factory created module with Name = %v, want %v", resultMod.Name, "test")
	}
}

func TestModuleOptionsCombined(t *testing.T) {
	provider := "provider"
	ctrl := func() {}
	mw := func() {}
	imported := New("imported")
	initFn := func() error { return nil }
	destroyFn := func() error { return nil }

	m := New(
		"test",
		Providers(provider),
		Controllers(ctrl),
		Middlewares(mw),
		Imports(imported),
		OnModuleInit(initFn),
		OnModuleDestroy(destroyFn),
	)

	if m.Name != "test" {
		t.Errorf("Module Name = %v, want %v", m.Name, "test")
	}
	if len(m.Providers) != 1 {
		t.Errorf("Module Providers length = %d, want 1", len(m.Providers))
	}
	if len(m.Controllers) != 1 {
		t.Errorf("Module Controllers length = %d, want 1", len(m.Controllers))
	}
	if len(m.Middlewares) != 1 {
		t.Errorf("Module Middlewares length = %d, want 1", len(m.Middlewares))
	}
	if len(m.Imports) != 1 {
		t.Errorf("Module Imports length = %d, want 1", len(m.Imports))
	}
	if len(m.OnInit) != 1 {
		t.Errorf("Module OnInit length = %d, want 1", len(m.OnInit))
	}
	if len(m.OnDestroy) != 1 {
		t.Errorf("Module OnDestroy length = %d, want 1", len(m.OnDestroy))
	}
}

func TestDynamicModule(t *testing.T) {
	factory := func(opts ...any) Module {
		return New("dynamic", Providers(opts...))
	}

	dm := &DynamicModule{
		Factory: factory,
		Options: []any{"opt1", "opt2"},
	}

	result := dm.Factory(dm.Options...)
	if result.Name != "dynamic" {
		t.Errorf("DynamicModule Factory Name = %v, want %v", result.Name, "dynamic")
	}
	if len(result.Providers) != 2 {
		t.Errorf("DynamicModule Factory Providers length = %d, want 2", len(result.Providers))
	}
}
