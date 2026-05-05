package container

import (
	"errors"
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

type testDoer interface{ Do() string }
type testDoerImpl struct{}

func (testDoerImpl) Do() string { return "done" }

type testGreeter interface{ Greet() string }
type testGreeterA struct{}
type testGreeterB struct{}

func (testGreeterA) Greet() string { return "hello-a" }
func (testGreeterB) Greet() string { return "hello-b" }

func TestContainerProvideAndResolve(t *testing.T) {
	c := New()
	typ := reflect.TypeOf((*testService)(nil))
	c.Register(typ, NewEntry(nil, &testService{name: "test"}, nil, false, false, nil))

	entry := c.providers[typ]
	if entry.eager == nil {
		t.Fatal("expected provider to be registered")
	}
}

func TestResolveValue(t *testing.T) {
	c := New()
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(nil, &testService{name: "resolved"}, nil, false, false, nil))

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
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, false, false, nil))

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
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, true, false, nil))

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
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, false, false, nil))

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
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(nil, &testService{name: "base"}, nil, false, false, nil))

	// Register wrapper with dependency on testService
	factory := func(args []reflect.Value) (any, error) {
		return &testWrapper{svc: args[0].Interface().(*testService)}, nil
	}
	c.Register(reflect.TypeOf((*testWrapper)(nil)), NewEntry(factory, nil, []reflect.Type{
		reflect.TypeOf((*testService)(nil)),
	}, false, false, nil))

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
	}, false, false, nil))

	// Register B depends on A
	factoryB := func(args []reflect.Value) (any, error) {
		return &testServiceB{a: args[0].Interface().(*testServiceA)}, nil
	}
	c.Register(reflect.TypeOf((*testServiceB)(nil)), NewEntry(factoryB, nil, []reflect.Type{
		reflect.TypeOf((*testServiceA)(nil)),
	}, false, false, nil))

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on circular dependency")
		}
	}()

	Resolve[*testServiceA](c)
}

func TestDuplicateProvider(t *testing.T) {
	c := New()

	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(nil, &testService{name: "first"}, nil, false, false, nil))

	// Duplicate provider should be ignored (no panic)
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(nil, &testService{name: "second"}, nil, false, false, nil))

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
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, false, false, nil))

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

func TestResolveInterfaceTypeDirectKey(t *testing.T) {
	c := New()
	doerType := reflect.TypeOf((*testDoer)(nil)).Elem()
	c.Register(doerType, NewEntry(nil, testDoerImpl{}, nil, false, false, nil))

	result := Resolve[testDoer](c)
	if result.Do() != "done" {
		t.Fatalf("expected 'done', got %s", result.Do())
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
	c.Register(reflect.TypeOf((*testService)(nil)), NewEntry(factory, nil, nil, true, false, nil))

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

func TestResolveByInterface_FallbackScan(t *testing.T) {
	c := New()
	implType := reflect.TypeOf(testGreeterA{})
	c.Register(implType, NewEntry(nil, testGreeterA{}, nil, false, false, nil))

	greeterType := reflect.TypeOf((*testGreeter)(nil)).Elem()
	result, err := c.resolve(greeterType, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(testGreeter).Greet() != "hello-a" {
		t.Fatalf("expected 'hello-a', got %s", result.(testGreeter).Greet())
	}
}

func TestResolveByInterface_AmbiguousReturnsError(t *testing.T) {
	c := New()
	c.Register(reflect.TypeOf(testGreeterA{}), NewEntry(nil, testGreeterA{}, nil, false, false, nil))
	c.Register(reflect.TypeOf(testGreeterB{}), NewEntry(nil, testGreeterB{}, nil, false, false, nil))

	greeterType := reflect.TypeOf((*testGreeter)(nil)).Elem()
	_, err := c.resolve(greeterType, nil)
	if err == nil {
		t.Fatal("expected error for ambiguous dependency")
	}

	var ambig *ErrAmbiguousDependency
	if !errors.As(err, &ambig) {
		t.Fatalf("expected ErrAmbiguousDependency, got %T: %v", err, err)
	}
	if len(ambig.Implementors) != 2 {
		t.Fatalf("expected 2 implementors, got %d: %v", len(ambig.Implementors), ambig.Implementors)
	}
}

func TestResolveByInterface_CachedAfterFirst(t *testing.T) {
	c := New()
	counter := atomic.Int32{}
	factory := func(args []reflect.Value) (any, error) {
		counter.Add(1)
		return testGreeterA{}, nil
	}
	c.Register(reflect.TypeOf(testGreeterA{}), NewEntry(factory, nil, nil, false, false, nil))

	greeterType := reflect.TypeOf((*testGreeter)(nil)).Elem()

	if _, err := c.resolve(greeterType, nil); err != nil {
		t.Fatalf("first resolve failed: %v", err)
	}
	if _, err := c.resolve(greeterType, nil); err != nil {
		t.Fatalf("second resolve failed: %v", err)
	}

	if counter.Load() != 1 {
		t.Fatalf("expected factory called once (cached), got %d", counter.Load())
	}
}

// testSvcA and testSvcB are used in TestBuildPreservesCause to avoid conflicts.
type testSvcA struct{}
type testSvcB struct{}

func TestBuildPreservesCause(t *testing.T) {
	c := New()
	bType := reflect.TypeOf(testSvcB{})
	aType := reflect.TypeOf(testSvcA{})

	// Register serviceA with a factory that requires serviceB (not registered)
	c.Register(aType, NewEntry(
		func(args []reflect.Value) (any, error) { return testSvcA{}, nil },
		nil,
		[]reflect.Type{bType},
		false, false, nil,
	))

	_, err := c.resolve(aType, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var missing *ErrMissingDependency
	if !errors.As(err, &missing) {
		t.Fatalf("expected ErrMissingDependency, got %T: %v", err, err)
	}
	if missing.Type != bType.String() {
		t.Fatalf("expected missing type %s, got %s", bType.String(), missing.Type)
	}
	if missing.RequiredBy != aType.String() {
		t.Fatalf("expected required by %s, got %s", aType.String(), missing.RequiredBy)
	}
}

func TestResolveByType_ReturnsError(t *testing.T) {
	c := New()
	typ := reflect.TypeOf(testSvcA{})
	// Not registered — should return error
	val, err := ResolveByType(c, typ)
	if err == nil {
		t.Fatal("expected error from ResolveByType when type not registered")
	}
	if val != nil {
		t.Fatalf("expected nil value, got %v", val)
	}
}
