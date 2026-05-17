package http

import (
	"context"
	"io"
	nethttp "net/http"
	"strings"
	"testing"

	"github.com/linkeunid/ligo/internal/di"
)

type mockBindAdapter struct {
	store   map[string]any
	bindErr error
}

func newMockBindAdapter() *mockBindAdapter {
	return &mockBindAdapter{store: map[string]any{}}
}

func (m *mockBindAdapter) Bind(any) error                     { return m.bindErr }
func (m *mockBindAdapter) Set(key string, val any)            { m.store[key] = val }
func (m *mockBindAdapter) Get(key string) any                 { return m.store[key] }
func (m *mockBindAdapter) Param(string) string                { return "" }
func (m *mockBindAdapter) Query(string) string                { return "" }
func (m *mockBindAdapter) BindQuery(any) error                { return nil }
func (m *mockBindAdapter) JSON(int, any) error                { return nil }
func (m *mockBindAdapter) String(int, string) error           { return nil }
func (m *mockBindAdapter) Stream(io.Reader) error             { return nil }
func (m *mockBindAdapter) Request() *nethttp.Request          { return nil }
func (m *mockBindAdapter) Response() nethttp.ResponseWriter   { return nil }
func (m *mockBindAdapter) RequestContext() context.Context    { return context.Background() }
func (m *mockBindAdapter) SetRequestContainer(*di.Container)  {}
func (m *mockBindAdapter) GetRequestContainer() *di.Container { return nil }

func newMockBindCtx() (*Context, *mockBindAdapter) {
	a := newMockBindAdapter()
	return NewContext(a), a
}

type testInput struct {
	Name string
}

func TestValidationPipe_StoresValidatedBody(t *testing.T) {
	ctx, a := newMockBindCtx()
	pipe := ValidationPipe[testInput](nil)

	if err := pipe(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.store[ValidatedBodyKey] == nil {
		t.Fatal("expected validated body in context, got nil")
	}
	if _, ok := a.store[ValidatedBodyKey].(*testInput); !ok {
		t.Fatalf("expected *testInput, got %T", a.store[ValidatedBodyKey])
	}
}

func TestValidatedBody_ReturnsStoredValue(t *testing.T) {
	ctx, _ := newMockBindCtx()
	pipe := ValidationPipe[testInput](nil)
	if err := pipe(ctx); err != nil {
		t.Fatalf("setup pipe failed: %v", err)
	}

	got := ValidatedBody[testInput](ctx)
	if got == nil {
		t.Fatal("ValidatedBody returned nil")
	}
}

func TestValidatedBody_PanicsWhenMissing(t *testing.T) {
	ctx, _ := newMockBindCtx()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if !strings.Contains(msg, "ValidationPipe") {
			t.Errorf("panic message should mention ValidationPipe, got: %s", msg)
		}
	}()

	ValidatedBody[testInput](ctx)
}
