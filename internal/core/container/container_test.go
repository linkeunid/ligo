package container

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

type testService struct {
	name string
}

type testServiceA struct {
	b *testServiceB
}

type testServiceB struct {
	a *testServiceA
}

type testWrapper struct {
	svc *testService
}

func TestContainerProvideAndResolve(t *testing.T) {
	c := New()
	typ := reflect.TypeOf((*testService)(nil))
	c.Register(typ, NewEntry(nil, &testService{name: "test"}, nil, false, false))

	entry := c.providers[typ]
	if entry.eager == nil {
		t.Fatal("expected provider to be registered")
	}
}

func TestResolveValue(t *testing.T) {
	c := New()
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(nil, &testService{name: "resolved"}, nil, false, false))

	svc := Resolve[*testService](c)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.name != "resolved" {
		t.Fatalf("expected 'resolved', got %s", svc.name)
	}
}

func TestResolveMissing(t *testing.T) {
	c := New()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on missing dependency")
		}
	}()

	Resolve[*testService](c)
}

func TestResolveFactory(t *testing.T) {
	c := New()

	factory := func(args []reflect.Value) (any, error) {
		return &testService{name: "factory"}, nil
	}
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, false, false))

	svc := Resolve[*testService](c)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.name != "factory" {
		t.Fatalf("expected 'factory', got %s", svc.name)
	}
}

func TestResolveTransient(t *testing.T) {
	c := New()

	counter := atomic.Int32{}
	factory := func(args []reflect.Value) (any, error) {
		counter.Add(1)
		return &testService{name: "transient"}, nil
	}
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, true, false))

	svc1 := Resolve[*testService](c)
	svc2 := Resolve[*testService](c)

	if svc1 == svc2 {
		t.Fatal("expected different instances for transient")
	}
	if counter.Load() != 2 {
		t.Fatalf("expected factory called 2 times, got %d", counter.Load())
	}
}

func TestResolveSingleton(t *testing.T) {
	c := New()

	counter := atomic.Int32{}
	factory := func(args []reflect.Value) (any, error) {
		counter.Add(1)
		return &testService{name: "singleton"}, nil
	}
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, false, false))

	svc1 := Resolve[*testService](c)
	svc2 := Resolve[*testService](c)

	if svc1 != svc2 {
		t.Fatal("expected same instance for singleton")
	}
	if counter.Load() != 1 {
		t.Fatalf("expected factory called 1 time, got %d", counter.Load())
	}
}

func TestAutoInject(t *testing.T) {
	c := New()

	// Register base service
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(nil, &testService{name: "base"}, nil, false, false))

	// Register wrapper with dependency on testService
	factory := func(args []reflect.Value) (any, error) {
		return &testWrapper{svc: args[0].Interface().(*testService)}, nil
	}
	c.Register(reflect.TypeOf((*testWrapper)(nil)), NewEntry(factory, nil, []reflect.Type{
		reflect.TypeOf((*testService)(nil)),
	}, false, false))

	wrapper := Resolve[*testWrapper](c)
	if wrapper.svc == nil {
		t.Fatal("expected injected service")
	}
	if wrapper.svc.name != "base" {
		t.Fatalf("expected 'base', got %s", wrapper.svc.name)
	}
}

func TestCircularDependency(t *testing.T) {
	c := New()

	// Register A depends on B
	factoryA := func(args []reflect.Value) (any, error) {
		return &testServiceA{b: args[0].Interface().(*testServiceB)}, nil
	}
	c.Register(reflect.TypeOf((*testServiceA)(nil)), NewEntry(factoryA, nil, []reflect.Type{
		reflect.TypeOf((*testServiceB)(nil)),
	}, false, false))

	// Register B depends on A
	factoryB := func(args []reflect.Value) (any, error) {
		return &testServiceB{a: args[0].Interface().(*testServiceA)}, nil
	}
	c.Register(reflect.TypeOf((*testServiceB)(nil)), NewEntry(factoryB, nil, []reflect.Type{
		reflect.TypeOf((*testServiceA)(nil)),
	}, false, false))

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on circular dependency")
		}
	}()

	Resolve[*testServiceA](c)
}

func TestDuplicateProvider(t *testing.T) {
	c := New()

	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(nil, &testService{name: "first"}, nil, false, false))

	// Duplicate provider should be ignored (no panic)
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(nil, &testService{name: "second"}, nil, false, false))

	// Verify the first provider is still used
	svc := Resolve[*testService](c)
	if svc.name != "first" {
		t.Fatalf("expected first provider to be used, got %s", svc.name)
	}
}

// TestConcurrentResolve verifies thread-safe singleton creation.
func TestConcurrentResolve(t *testing.T) {
	c := New()
	counter := atomic.Int32{}
	factory := func(args []reflect.Value) (any, error) {
		counter.Add(1)
		return &testService{name: "concurrent"}, nil
	}
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, false, false))

	// Resolve concurrently from 10 goroutines
	var wg sync.WaitGroup
	results := make([]*testService, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = Resolve[*testService](c)
		}(i)
	}
	wg.Wait()

	// All results should be the same singleton instance
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Fatal("expected same singleton instance across concurrent resolves")
		}
	}
	// Factory should only be called once
	if counter.Load() != 1 {
		t.Fatalf("expected factory called 1 time, got %d", counter.Load())
	}
}

// TestConcurrentTransient verifies concurrent transient resolves.
func TestConcurrentTransient(t *testing.T) {
	c := New()
	counter := atomic.Int32{}
	factory := func(args []reflect.Value) (any, error) {
		n := counter.Add(1)
		return &testService{name: fmt.Sprintf("instance-%d", n)}, nil
	}
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, true, false))

	var wg sync.WaitGroup
	results := make([]*testService, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = Resolve[*testService](c)
		}(i)
	}
	wg.Wait()

	// All results should be different instances (transient)
	instances := make(map[*testService]bool)
	for i := 0; i < len(results); i++ {
		if results[i] == nil {
			t.Fatal("expected non-nil instance")
		}
		if instances[results[i]] {
			t.Fatalf("expected unique instances for transient")
		}
		instances[results[i]] = true
	}
	// Factory should be called 10 times
	if counter.Load() != 10 {
		t.Fatalf("expected factory called 10 times, got %d", counter.Load())
	}
}
