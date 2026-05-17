package testing

import (
	"errors"
	"testing"

	httpifc "github.com/linkeunid/ligo/internal/http"
)

// Compile-time check duplicates the one in mocks.go but keeps it visible from
// tests so a CI change cannot delete the assertion silently.
var _ httpifc.Context = (*MockContext)(nil)

type bindTarget struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestMockContext_BindCopiesStashedBody(t *testing.T) {
	m := NewMockContext()
	m.SetBody(bindTarget{Name: "alice", Age: 30})

	var got bindTarget
	if err := m.Bind(&got); err != nil {
		t.Fatalf("bind err: %v", err)
	}
	if got.Name != "alice" || got.Age != 30 {
		t.Errorf("bind copied wrong values: %+v", got)
	}
}

func TestMockContext_BindReturnsInjectedError(t *testing.T) {
	m := NewMockContext()
	want := errors.New("nope")
	m.WithBindError(want)

	var got bindTarget
	if err := m.Bind(&got); !errors.Is(err, want) {
		t.Errorf("expected injected err, got %v", err)
	}
}

func TestMockContext_BindDefaultIsNoop(t *testing.T) {
	m := NewMockContext()
	var got bindTarget
	if err := m.Bind(&got); err != nil {
		t.Errorf("default Bind should be nil err, got %v", err)
	}
	if got.Name != "" || got.Age != 0 {
		t.Errorf("default Bind should leave target zero, got %+v", got)
	}
}

func TestMockContext_BindQueryCopiesStashed(t *testing.T) {
	m := NewMockContext()
	m.SetQueryBody(bindTarget{Name: "bob"})

	var got bindTarget
	if err := m.BindQuery(&got); err != nil {
		t.Fatalf("bindquery err: %v", err)
	}
	if got.Name != "bob" {
		t.Errorf("bindquery copied wrong: %+v", got)
	}
}

func TestMockContext_BindQueryReturnsInjectedError(t *testing.T) {
	m := NewMockContext()
	want := errors.New("query nope")
	m.WithBindQueryError(want)

	var got bindTarget
	if err := m.BindQuery(&got); !errors.Is(err, want) {
		t.Errorf("expected injected err, got %v", err)
	}
}

func TestMockContext_ImATeapotReturnsNil(t *testing.T) {
	m := NewMockContext()
	if err := m.ImATeapot("yes"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
