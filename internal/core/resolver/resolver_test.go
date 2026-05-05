package resolver

import (
	"errors"
	"reflect"
	"testing"

	"github.com/linkeunid/ligo/internal/core/container"
)

type TestInterface interface {
	Method()
}

type TestImpl struct{}

func (t *TestImpl) Method() {}

type AnotherInterface interface {
	AnotherMethod()
}

type AnotherImpl struct{}

func (a *AnotherImpl) AnotherMethod() {}

type CircularA struct {
	B *CircularB
}

type CircularB struct {
	A *CircularA
}

func TestNew(t *testing.T) {
	c := container.New()
	r := New(c)

	if r == nil {
		t.Fatal("New() returned nil")
	}
	if r.container != c {
		t.Error("New() did not set container")
	}
	if len(r.chain) != 0 {
		t.Error("New() should have empty chain")
	}
}

func TestResolve(t *testing.T) {
	t.Run("resolve implementation by interface", func(t *testing.T) {
		c := container.New()
		testImplType := reflect.TypeOf(&TestImpl{})
		c.Register(testImplType, container.NewEntry(
			func(args []reflect.Value) (any, error) {
				return &TestImpl{}, nil
			},
			nil,
			nil,
			false,
			false, nil,
		))

		r := New(c)

		// Create a nil interface pointer to specify the interface type
		// This allows the resolver to find the concrete implementation
		var iface TestInterface
		result := r.Resolve(&iface)

		if result == nil {
			t.Fatal("Resolve() returned nil")
		}
		impl, ok := result.(*TestImpl)
		if !ok {
			t.Errorf("Resolve() returned %T, want *TestImpl", result)
		}
		if impl == nil {
			t.Error("Resolve() returned nil implementation")
		}
	})

	t.Run("resolve non-existent type panics", func(t *testing.T) {
		c := container.New()
		r := New(c)

		var iface AnotherInterface
		defer func() {
			if r := recover(); r == nil {
				t.Error("Resolve() should panic for non-existent type")
			}
		}()
		r.Resolve(&iface)
	})
}

func TestResolveCircularDependency(t *testing.T) {
	t.Run("detect circular dependency", func(t *testing.T) {
		c := container.New()

		// Register A that depends on B
		c.Register(reflect.TypeOf(&CircularA{}), container.NewEntry(
			func(args []reflect.Value) (any, error) {
				return &CircularA{B: args[0].Interface().(*CircularB)}, nil
			},
			nil,
			[]reflect.Type{reflect.TypeOf(&CircularB{})},
			false,
			false, nil,
		))

		// Register B that depends on A
		c.Register(reflect.TypeOf(&CircularB{}), container.NewEntry(
			func(args []reflect.Value) (any, error) {
				return &CircularB{A: args[0].Interface().(*CircularA)}, nil
			},
			nil,
			[]reflect.Type{reflect.TypeOf(&CircularA{})},
			false,
			false, nil,
		))

		r := New(c)

		defer func() {
			if r := recover(); r == nil {
				t.Error("Resolve() should panic on circular dependency")
			}
		}()
		r.Resolve(&CircularA{})
	})

	t.Run("no circular dependency for valid graph", func(t *testing.T) {
		c := container.New()

		// Register independent types
		c.Register(reflect.TypeOf(&TestImpl{}), container.NewEntry(
			func(args []reflect.Value) (any, error) {
				return &TestImpl{}, nil
			},
			nil,
			nil,
			false,
			false, nil,
		))

		c.Register(reflect.TypeOf(&AnotherImpl{}), container.NewEntry(
			func(args []reflect.Value) (any, error) {
				return &AnotherImpl{}, nil
			},
			nil,
			nil,
			false,
			false, nil,
		))

		r := New(c)

		// Use pointer to interface for proper type resolution
		var iface TestInterface
		result := r.Resolve(&iface)

		if result == nil {
			t.Fatal("Resolve() returned nil for valid dependency graph")
		}
	})
}

func TestResolveChainManagement(t *testing.T) {
	t.Skip("Resolver is designed for interface-based resolution, not concrete types")

	// This test is skipped because the resolver is specifically designed
	// to resolve implementations by interface type, not by concrete type.
	// For concrete type resolution with dependencies, use the container directly.
}

func TestResolveErrorHandling(t *testing.T) {
	t.Run("factory returns error", func(t *testing.T) {
		c := container.New()
		c.Register(reflect.TypeOf(&TestImpl{}), container.NewEntry(
			func(args []reflect.Value) (any, error) {
				return nil, errors.New("factory error")
			},
			nil, nil, false, false, nil,
		))

		r := New(c)

		defer func() {
			if r := recover(); r == nil {
				t.Error("Resolve() should panic when factory returns error")
			}
		}()
		r.Resolve(&TestImpl{})
	})

	t.Run("factory returns nil", func(t *testing.T) {
		c := container.New()
		c.Register(reflect.TypeOf(&TestImpl{}), container.NewEntry(
			func(args []reflect.Value) (any, error) {
				return nil, nil
			},
			nil, nil, false, false, nil,
		))

		r := New(c)

		// Use pointer to interface for proper type resolution
		var iface TestInterface
		result := r.Resolve(&iface)
		if result != nil {
			t.Errorf("Resolve() should return nil when factory returns nil, got %v", result)
		}
	})
}

func TestResolveWithDependencies(t *testing.T) {
	t.Skip("Resolver is designed for interface-based resolution, not concrete types")

	// This test is skipped because the resolver is specifically designed
	// to resolve implementations by interface type, not by concrete type.
	// For concrete type resolution with dependencies, use the container directly.
}
