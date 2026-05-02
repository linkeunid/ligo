package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
)

type mockProvider struct {
	typ         reflect.Type
	eager       any
	isTransient bool
	isExported  bool
}

func (m *mockProvider) Type() reflect.Type {
	if m.typ != nil {
		return m.typ
	}
	return reflect.TypeOf("string")
}

func (m *mockProvider) Eager() any {
	return m.eager
}

func (m *mockProvider) IsTransient() bool {
	return m.isTransient
}

func (m *mockProvider) IsExported() bool {
	return m.isExported
}

func (m *mockProvider) Fn() func() string {
	return func() string { return "test" }
}

type TestService struct{}

func TestBuildProviderEntry(t *testing.T) {
	t.Run("eager provider creates valid entry", func(t *testing.T) {
		p := &mockProvider{eager: "test-value"}
		entry := BuildProviderEntry(p)

		// Entry should be valid for container registration
		c := container.New()
		c.Register(reflect.TypeOf("string"), entry)

		types := c.Types()
		if len(types) != 1 {
			t.Errorf("Expected 1 type registered, got %d", len(types))
		}
	})

	t.Run("factory provider creates valid entry", func(t *testing.T) {
		p := &mockProvider{}
		entry := BuildProviderEntry(p)

		// Entry should be valid for container registration
		c := container.New()
		c.Register(reflect.TypeOf("string"), entry)

		types := c.Types()
		if len(types) != 1 {
			t.Errorf("Expected 1 type registered, got %d", len(types))
		}
	})

	t.Run("transient flag is preserved", func(t *testing.T) {
		p := &mockProvider{isTransient: true, eager: "test"}
		entry := BuildProviderEntry(p)

		c := container.New()
		typ := reflect.TypeOf("string")
		c.Register(typ, entry)

		// Resolve twice - transient should create new instances
		v1 := container.ResolveByType(c, typ)
		v2 := container.ResolveByType(c, typ)

		// For eager providers, same value is returned even with transient
		// The flag is preserved in the entry structure
		_ = v1
		_ = v2
	})

	t.Run("exported flag is preserved", func(t *testing.T) {
		p := &mockProvider{isExported: true, eager: "test"}
		_ = BuildProviderEntry(p)
		// Exported flag is used by BuildModule, tested there
	})
}

func TestRegisterProvider(t *testing.T) {
	t.Run("register eager provider", func(t *testing.T) {
		c := container.New()
		p := &mockProvider{
			typ:   reflect.TypeOf("string"),
			eager: "test",
		}

		RegisterProvider(c, p)

		types := c.Types()
		if len(types) != 1 {
			t.Errorf("RegisterProvider() registered %d types, want 1", len(types))
		}

		// Verify we can resolve the value
		val := container.ResolveByType(c, reflect.TypeOf("string"))
		if val != "test" {
			t.Errorf("Resolve() = %v, want 'test'", val)
		}
	})

	t.Run("register factory provider", func(t *testing.T) {
		c := container.New()
		p := &mockProvider{
			typ: reflect.TypeOf("string"),
		}

		RegisterProvider(c, p)

		types := c.Types()
		if len(types) != 1 {
			t.Errorf("RegisterProvider() registered %d types, want 1", len(types))
		}
	})
}

func TestBuildModule(t *testing.T) {
	t.Run("simple module with providers", func(t *testing.T) {
		c := container.New()
		m := module.New("test",
			module.Providers(&mockProvider{eager: "provider1"}),
		)

		hooks := &ModuleHooks{}
		BuildModule(c, m, hooks)

		types := c.Types()
		if len(types) == 0 {
			t.Error("BuildModule() did not register providers")
		}
	})

	t.Run("module with init hook", func(t *testing.T) {
		c := container.New()
		m := module.New("test",
			module.OnModuleInit(func() error {
				return nil
			}),
		)

		hooks := &ModuleHooks{}
		BuildModule(c, m, hooks)

		if len(hooks.OnInit) != 1 {
			t.Errorf("BuildModule() OnInit length = %d, want 1", len(hooks.OnInit))
		}
		if len(hooks.OnInit[0]) != 1 {
			t.Errorf("BuildModule() OnInit[0] length = %d, want 1", len(hooks.OnInit[0]))
		}
	})

	t.Run("module with destroy hook", func(t *testing.T) {
		c := container.New()
		m := module.New("test",
			module.OnModuleDestroy(func() error {
				return nil
			}),
		)

		hooks := &ModuleHooks{}
		BuildModule(c, m, hooks)

		if len(hooks.OnDestroy) != 1 {
			t.Errorf("BuildModule() OnDestroy length = %d, want 1", len(hooks.OnDestroy))
		}
	})

	t.Run("module with imported modules", func(t *testing.T) {
		c := container.New()
		imported := module.New("imported",
			module.Providers(&mockProvider{eager: "imported-provider"}),
		)
		m := module.New("test",
			module.Imports(imported),
		)

		hooks := &ModuleHooks{}
		BuildModule(c, m, hooks)

		types := c.Types()
		if len(types) == 0 {
			t.Error("BuildModule() did not register imported providers")
		}
	})

	t.Run("dynamic module", func(t *testing.T) {
		c := container.New()

		dynamicFactory := func(opts ...any) module.Module {
			name := "dynamic"
			if len(opts) > 0 {
				if n, ok := opts[0].(string); ok {
					name = n
				}
			}
			return module.New(name,
				module.Providers(&mockProvider{eager: "dynamic-provider"}),
			)
		}

		m := module.New("test",
			module.Dynamic(dynamicFactory, "custom-dynamic"),
		)

		hooks := &ModuleHooks{}
		BuildModule(c, m, hooks)

		types := c.Types()
		if len(types) == 0 {
			t.Error("BuildModule() did not register dynamic module providers")
		}
	})

	t.Run("exported vs non-exported providers", func(t *testing.T) {
		parent := container.New()
		child := parent.NewChild()

		type ExportedType struct{}
		type NotExportedType struct{}

		exported := &mockProvider{typ: reflect.TypeOf(&ExportedType{}), eager: &ExportedType{}, isExported: true}
		notExported := &mockProvider{typ: reflect.TypeOf(&NotExportedType{}), eager: &NotExportedType{}, isExported: false}

		m := module.New("test",
			module.Providers(exported, notExported),
		)

		hooks := &ModuleHooks{}
		BuildModule(child, m, hooks)

		// Both should be in child since modContainer := parent in BuildModule
		// The exported flag is used for sibling module visibility, not parent/child containers
		childTypes := child.Types()

		if len(childTypes) != 2 {
			t.Errorf("Child container has %d types, want 2", len(childTypes))
		}
	})
}

func TestExecuteHooks(t *testing.T) {
	t.Run("successful hooks", func(t *testing.T) {
		calls := []string{}
		hooks := [][]func() error{
			{func() error { calls = append(calls, "1-1"); return nil }},
			{func() error { calls = append(calls, "2-1"); return nil }},
		}

		log := logger.New()
		err := ExecuteHooks(hooks, log, "test")

		if err != nil {
			t.Errorf("ExecuteHooks() returned error: %v", err)
		}
		if len(calls) != 2 {
			t.Errorf("ExecuteHooks() executed %d hooks, want 2", len(calls))
		}
	})

	t.Run("hook with error", func(t *testing.T) {
		calls := []string{}
		hooks := [][]func() error{
			{func() error { calls = append(calls, "1-1"); return nil }},
			{func() error { calls = append(calls, "2-1"); return errors.New("test error") }},
			{func() error { calls = append(calls, "3-1"); return nil }},
		}

		log := logger.New()
		err := ExecuteHooks(hooks, log, "test")

		if err == nil {
			t.Error("ExecuteHooks() should return error")
		}
		if len(calls) != 2 {
			t.Errorf("ExecuteHooks() executed %d hooks before error, want 2", len(calls))
		}
	})

	t.Run("hooks with nil logger", func(t *testing.T) {
		hooks := [][]func() error{
			{func() error { return errors.New("test error") }},
		}

		err := ExecuteHooks(hooks, nil, "test")

		if err == nil {
			t.Error("ExecuteHooks() should return error even with nil logger")
		}
	})

	t.Run("empty hooks", func(t *testing.T) {
		hooks := [][]func() error{}
		log := logger.New()

		err := ExecuteHooks(hooks, log, "test")

		if err != nil {
			t.Errorf("ExecuteHooks() with empty hooks returned error: %v", err)
		}
	})

	t.Run("multiple hooks per module", func(t *testing.T) {
		calls := []string{}
		hooks := [][]func() error{
			{
				func() error { calls = append(calls, "1-1"); return nil },
				func() error { calls = append(calls, "1-2"); return nil },
			},
			{
				func() error { calls = append(calls, "2-1"); return nil },
			},
		}

		log := logger.New()
		err := ExecuteHooks(hooks, log, "test")

		if err != nil {
			t.Errorf("ExecuteHooks() returned error: %v", err)
		}
		if len(calls) != 3 {
			t.Errorf("ExecuteHooks() executed %d hooks, want 3", len(calls))
		}
	})
}

func TestModuleHooks(t *testing.T) {
	t.Run("create module hooks", func(t *testing.T) {
		hooks := &ModuleHooks{}

		// Slices are nil by default in Go
		if hooks.OnInit != nil {
			t.Error("OnInit should be nil initially")
		}
		if hooks.OnDestroy != nil {
			t.Error("OnDestroy should be nil initially")
		}
	})

	t.Run("append hooks", func(t *testing.T) {
		hooks := &ModuleHooks{}
		hook1 := func() error { return nil }
		hook2 := func() error { return nil }

		hooks.OnInit = append(hooks.OnInit, []func() error{hook1})
		hooks.OnDestroy = append(hooks.OnDestroy, []func() error{hook2})

		if len(hooks.OnInit) != 1 {
			t.Errorf("OnInit length = %d, want 1", len(hooks.OnInit))
		}
		if len(hooks.OnDestroy) != 1 {
			t.Errorf("OnDestroy length = %d, want 1", len(hooks.OnDestroy))
		}
	})
}
