package di

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

type (
	testDoer     interface{ Do() string }
	testDoerImpl struct{}
)

func (testDoerImpl) Do() string { return "done" }

type (
	testGreeter  interface{ Greet() string }
	testGreeterA struct{}
	testGreeterB struct{}
)

func (testGreeterA) Greet() string { return "hello-a" }
func (testGreeterB) Greet() string { return "hello-b" }

func TestContainerProvideAndResolve(t *testing.T) {
	c := New()
	typ := reflect.TypeFor[*testService]()
	c.Register(typ, NewEntry(nil, &testService{name: "test"}, nil, false, false, nil))

	entry := c.providers[typ]
	if entry.eager == nil {
		t.Fatal("expected provider to be registered")
	}
}

func TestResolveValue(t *testing.T) {
	c := New()
	c.Register(reflect.TypeFor[*testService](), NewEntry(nil, &testService{name: "resolved"}, nil, false, false, nil))

	svc := MustResolve[*testService](c)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.name != "resolved" {
		t.Fatalf("expected 'resolved', got %s", svc.name)
	}
}

func TestResolveMissing(t *testing.T) {
	c := New()

	if _, err := Resolve[*testService](c); err == nil {
		t.Fatal("expected error on missing dependency")
	}
}

func TestMustResolveMissingPanics(t *testing.T) {
	c := New()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on missing dependency")
		}
	}()

	MustResolve[*testService](c)
}

func TestResolveFactory(t *testing.T) {
	c := New()

	factory := func(args []reflect.Value) (any, error) {
		return &testService{name: "factory"}, nil
	}
	c.Register(reflect.TypeFor[*testService](), NewEntry(factory, nil, nil, false, false, nil))

	svc := MustResolve[*testService](c)
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
	c.Register(reflect.TypeFor[*testService](), NewEntry(factory, nil, nil, true, false, nil))

	svc1 := MustResolve[*testService](c)
	svc2 := MustResolve[*testService](c)

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
	c.Register(reflect.TypeFor[*testService](), NewEntry(factory, nil, nil, false, false, nil))

	svc1 := MustResolve[*testService](c)
	svc2 := MustResolve[*testService](c)

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
	c.Register(reflect.TypeFor[*testService](), NewEntry(nil, &testService{name: "base"}, nil, false, false, nil))

	// Register wrapper with dependency on testService
	factory := func(args []reflect.Value) (any, error) {
		return &testWrapper{svc: args[0].Interface().(*testService)}, nil
	}
	c.Register(reflect.TypeFor[*testWrapper](), NewEntry(factory, nil, []reflect.Type{
		reflect.TypeFor[*testService](),
	}, false, false, nil))

	wrapper := MustResolve[*testWrapper](c)
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
	c.Register(reflect.TypeFor[*testServiceA](), NewEntry(factoryA, nil, []reflect.Type{
		reflect.TypeFor[*testServiceB](),
	}, false, false, nil))

	// Register B depends on A
	factoryB := func(args []reflect.Value) (any, error) {
		return &testServiceB{a: args[0].Interface().(*testServiceA)}, nil
	}
	c.Register(reflect.TypeFor[*testServiceB](), NewEntry(factoryB, nil, []reflect.Type{
		reflect.TypeFor[*testServiceA](),
	}, false, false, nil))

	if _, err := Resolve[*testServiceA](c); err == nil {
		t.Fatal("expected error on circular dependency")
	}
}

func TestDuplicateProvider(t *testing.T) {
	c := New()

	c.Register(reflect.TypeFor[*testService](), NewEntry(nil, &testService{name: "first"}, nil, false, false, nil))

	// Duplicate provider should be ignored (no panic)
	c.Register(reflect.TypeFor[*testService](), NewEntry(nil, &testService{name: "second"}, nil, false, false, nil))

	// Verify the first provider is still used
	svc := MustResolve[*testService](c)
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
	c.Register(reflect.TypeFor[*testService](), NewEntry(factory, nil, nil, false, false, nil))

	// Resolve concurrently from 10 goroutines
	var wg sync.WaitGroup
	results := make([]*testService, 10)
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = MustResolve[*testService](c)
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
	doerType := reflect.TypeFor[testDoer]()
	c.Register(doerType, NewEntry(nil, testDoerImpl{}, nil, false, false, nil))

	result := MustResolve[testDoer](c)
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
	c.Register(reflect.TypeFor[*testService](), NewEntry(factory, nil, nil, true, false, nil))

	var wg sync.WaitGroup
	results := make([]*testService, 10)
	for i := range 10 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = MustResolve[*testService](c)
		}(i)
	}
	wg.Wait()

	// All results should be different instances (transient)
	instances := make(map[*testService]bool)
	for i := range results {
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
	implType := reflect.TypeFor[testGreeterA]()
	c.Register(implType, NewEntry(nil, testGreeterA{}, nil, false, false, nil))

	greeterType := reflect.TypeFor[testGreeter]()
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
	c.Register(reflect.TypeFor[testGreeterA](), NewEntry(nil, testGreeterA{}, nil, false, false, nil))
	c.Register(reflect.TypeFor[testGreeterB](), NewEntry(nil, testGreeterB{}, nil, false, false, nil))

	greeterType := reflect.TypeFor[testGreeter]()
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
	c.Register(reflect.TypeFor[testGreeterA](), NewEntry(factory, nil, nil, false, false, nil))

	greeterType := reflect.TypeFor[testGreeter]()

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
type (
	testSvcA struct{}
	testSvcB struct{}
)

func TestBuildPreservesCause(t *testing.T) {
	c := New()
	bType := reflect.TypeFor[testSvcB]()
	aType := reflect.TypeFor[testSvcA]()

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

type emptyEntryFixture struct{}

func TestBuild_EmptyEntryReturnsError(t *testing.T) {
	c := New()
	typ := reflect.TypeFor[*emptyEntryFixture]()
	c.Register(typ, NewEntry(nil, nil, nil, false, false, nil))

	_, err := Resolve[*emptyEntryFixture](c)
	if err == nil {
		t.Fatal("expected error for entry with neither eager nor factory, got nil")
	}
	var dierr *DIError
	if !errors.As(err, &dierr) {
		t.Fatalf("expected *DIError in chain, got %T (%v)", err, err)
	}
	if !errors.Is(err, errEntryEmpty) {
		t.Fatalf("expected errors.Is(err, errEntryEmpty), got %v", dierr.Cause)
	}
}

func TestResolveByType_ReturnsError(t *testing.T) {
	c := New()
	typ := reflect.TypeFor[testSvcA]()
	// Not registered — should return error
	val, err := ResolveByType(c, typ)
	if err == nil {
		t.Fatal("expected error from ResolveByType when type not registered")
	}
	if val != nil {
		t.Fatalf("expected nil value, got %v", val)
	}
}

func TestDIError_Unwrap(t *testing.T) {
	cause := errors.New("synthetic factory failure")
	err := &DIError{Type: "Foo", Cause: cause}

	if !errors.Is(err, cause) {
		t.Fatal("expected errors.Is to walk DIError → Cause")
	}
	if got := errors.Unwrap(err); !errors.Is(got, cause) {
		t.Fatalf("Unwrap = %v, want %v", got, cause)
	}
}

func TestErrMissingDependency_Format(t *testing.T) {
	cases := []struct {
		name string
		err  *ErrMissingDependency
		want string
	}{
		{
			name: "top-level resolve (no parent)",
			err:  &ErrMissingDependency{Type: "*Foo"},
			want: "ligo: missing dependency *Foo",
		},
		{
			name: "nested resolve (with parent)",
			err:  &ErrMissingDependency{Type: "*Foo", RequiredBy: "*Bar"},
			want: "ligo: missing dependency *Foo (required by *Bar)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuild_FactoryErrorCarriesRequiredBy(t *testing.T) {
	c := New()
	type Inner struct{}
	type Outer struct{ in *Inner }

	factoryErr := errors.New("inner construction failed")

	innerTyp := reflect.TypeFor[*Inner]()
	c.Register(innerTyp, NewEntry(
		func(args []reflect.Value) (any, error) { return nil, factoryErr },
		nil, nil, false, false, nil,
	))

	outerTyp := reflect.TypeFor[*Outer]()
	c.Register(outerTyp, NewEntry(
		func(args []reflect.Value) (any, error) {
			return &Outer{in: args[0].Interface().(*Inner)}, nil
		},
		nil, []reflect.Type{innerTyp}, false, false, nil,
	))

	_, err := Resolve[*Outer](c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var dierr *DIError
	if !errors.As(err, &dierr) {
		t.Fatalf("expected *DIError in chain, got %T (%v)", err, err)
	}
	if dierr.RequiredBy == "" {
		t.Errorf("DIError.RequiredBy is empty; want parent type name")
	}
	if !errors.Is(err, factoryErr) {
		t.Errorf("expected errors.Is(err, factoryErr); chain is %v", err)
	}
}

func TestNewChild_InheritsFromParent(t *testing.T) {
	parent := New()
	typ := reflect.TypeFor[*testService]()
	parent.Register(typ, NewEntry(nil, &testService{name: "parent-svc"}, nil, false, false, nil))

	child := parent.NewChild()
	got, err := Resolve[*testService](child)
	if err != nil {
		t.Fatalf("child resolve err: %v", err)
	}
	if got.name != "parent-svc" {
		t.Errorf("child resolved %q, want parent-svc", got.name)
	}
}

func TestNewChild_OverridesParent(t *testing.T) {
	parent := New()
	typ := reflect.TypeFor[*testService]()
	parent.Register(typ, NewEntry(nil, &testService{name: "parent-svc"}, nil, false, false, nil))

	child := parent.NewChild()
	child.Register(typ, NewEntry(nil, &testService{name: "child-svc"}, nil, false, false, nil))

	got, err := Resolve[*testService](child)
	if err != nil {
		t.Fatalf("child resolve err: %v", err)
	}
	if got.name != "child-svc" {
		t.Errorf("child override = %q, want child-svc", got.name)
	}

	// Parent still resolves original.
	parentGot, _ := Resolve[*testService](parent)
	if parentGot.name != "parent-svc" {
		t.Errorf("parent leaked child override = %q", parentGot.name)
	}
}

func TestErrMissingDependency_Unwrap(t *testing.T) {
	root := errors.New("root cause")
	e := &ErrMissingDependency{Type: "Foo", Cause: root}
	if !errors.Is(e, root) {
		t.Error("errors.Is did not traverse Unwrap chain")
	}
}

func TestErrCircularDependency_Error(t *testing.T) {
	e := &ErrCircularDependency{Chain: []string{"A", "B", "A"}}
	msg := e.Error()
	if msg == "" {
		t.Error("empty error message")
	}
	for _, want := range []string{"circular", "A", "B"} {
		if !contains(msg, want) {
			t.Errorf("missing %q in %q", want, msg)
		}
	}
}

func TestErrAmbiguousDependency_Error(t *testing.T) {
	e := &ErrAmbiguousDependency{Interface: "io.Reader", Implementors: []string{"*A", "*B"}}
	msg := e.Error()
	for _, want := range []string{"ambiguous", "io.Reader", "*A", "*B"} {
		if !contains(msg, want) {
			t.Errorf("missing %q in %q", want, msg)
		}
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
