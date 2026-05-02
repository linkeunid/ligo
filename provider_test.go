package ligo

import (
	"reflect"
	"testing"
)

type testService struct {
	name string
}

type testDoer interface{ Do() string }
type testDoerImpl struct{}

func (testDoerImpl) Do() string { return "done" }

func TestValueProvider(t *testing.T) {
	svc := &testService{}
	p := Value(svc)
	if p.Type() == nil {
		t.Fatal("expected non-nil type")
	}
}

func TestFactoryProvider(t *testing.T) {
	p := Factory[*testService](func() *testService {
		return &testService{}
	})
	if p.Type() == nil {
		t.Fatal("expected non-nil type")
	}
}

func TestTransientProvider(t *testing.T) {
	p := Transient[*testService](func() *testService {
		return &testService{}
	})
	if p.Type() == nil {
		t.Fatal("expected non-nil type")
	}
	if !p.transient {
		t.Fatal("expected transient to be true")
	}
}

func TestExportProvider(t *testing.T) {
	p := Export(Factory[*testService](func() *testService {
		return &testService{}
	}))
	if !p.exported {
		t.Fatal("expected exported to be true")
	}
}

func TestFactoryInterfaceType(t *testing.T) {
	p := Factory[testDoer](func() testDoer { return testDoerImpl{} })
	if p.Type() == nil {
		t.Fatal("Factory[InterfaceType] must not register nil type")
	}
	if p.Type().Kind() != reflect.Interface {
		t.Fatalf("expected Interface kind, got %s", p.Type().Kind())
	}
}

func TestValueInterfaceType(t *testing.T) {
	p := Value[testDoer](testDoerImpl{})
	if p.Type() == nil {
		t.Fatal("Value[InterfaceType] must not register nil type")
	}
	if p.Type().Kind() != reflect.Interface {
		t.Fatalf("expected Interface kind, got %s", p.Type().Kind())
	}
}
