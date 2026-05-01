package ligo

import (
	"testing"
)

type testService struct {
	name string
}

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
