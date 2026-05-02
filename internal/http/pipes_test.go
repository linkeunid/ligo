package http

import (
	nethttp "net/http"
	"strings"
	"testing"

	"github.com/linkeunid/ligo/internal/core/container"
)

type mockBindContext struct {
	mockContextBase
	store   map[string]any
	bindErr error
}

func newMockBindContext() *mockBindContext {
	return &mockBindContext{store: map[string]any{}}
}

func (m *mockBindContext) Bind(v any) error {
	return m.bindErr
}

func (m *mockBindContext) Set(key string, val any) { m.store[key] = val }
func (m *mockBindContext) Get(key string) any      { return m.store[key] }

type mockContextBase struct{}

func (mockContextBase) Param(_ string) string                    { return "" }
func (mockContextBase) JSON(_ int, _ any) error                  { return nil }
func (mockContextBase) String(_ int, _ string) error             { return nil }
func (mockContextBase) OK(_ any) error                           { return nil }
func (mockContextBase) Created(_ any) error                      { return nil }
func (mockContextBase) Accepted(_ any) error                     { return nil }
func (mockContextBase) NoContent() error                         { return nil }
func (mockContextBase) BadRequest(_ ...string) error                { return nil }
func (mockContextBase) Unauthorized(_ ...string) error              { return nil }
func (mockContextBase) Forbidden(_ ...string) error                 { return nil }
func (mockContextBase) NotFound(_ ...string) error                  { return nil }
func (mockContextBase) MethodNotAllowed(_ ...string) error          { return nil }
func (mockContextBase) NotAcceptable(_ ...string) error             { return nil }
func (mockContextBase) RequestTimeout(_ ...string) error            { return nil }
func (mockContextBase) Conflict(_ ...string) error                  { return nil }
func (mockContextBase) Gone(_ ...string) error                      { return nil }
func (mockContextBase) PreconditionFailed(_ ...string) error        { return nil }
func (mockContextBase) PayloadTooLarge(_ ...string) error           { return nil }
func (mockContextBase) UnsupportedMediaType(_ ...string) error      { return nil }
func (mockContextBase) UnprocessableEntity(_ ...string) error       { return nil }
func (mockContextBase) TooManyRequests(_ ...string) error           { return nil }
func (mockContextBase) ImATeapot(_ ...string) error                 { return nil }
func (mockContextBase) InternalServerError(_ ...string) error       { return nil }
func (mockContextBase) NotImplemented(_ ...string) error            { return nil }
func (mockContextBase) BadGateway(_ ...string) error                { return nil }
func (mockContextBase) ServiceUnavailable(_ ...string) error        { return nil }
func (mockContextBase) GatewayTimeout(_ ...string) error            { return nil }
func (mockContextBase) HTTPVersionNotSupported(_ ...string) error   { return nil }
func (mockContextBase) Header(_ string) string                   { return "" }
func (mockContextBase) Stream(_ any) error                       { return nil }
func (mockContextBase) Request() *nethttp.Request                           { return nil }
func (mockContextBase) Response() nethttp.ResponseWriter                    { return nil }
func (mockContextBase) SetRequestContainer(_ *container.Container)          {}
func (mockContextBase) GetRequestContainer() *container.Container           { return nil }

type testInput struct {
	Name string
}

func TestValidationPipe_StoresValidatedBody(t *testing.T) {
	ctx := newMockBindContext()
	pipe := ValidationPipe[testInput](nil)

	if err := pipe(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.store[ValidatedBodyKey] == nil {
		t.Fatal("expected validated body in context, got nil")
	}
	if _, ok := ctx.store[ValidatedBodyKey].(*testInput); !ok {
		t.Fatalf("expected *testInput, got %T", ctx.store[ValidatedBodyKey])
	}
}

func TestValidatedBody_ReturnsStoredValue(t *testing.T) {
	ctx := newMockBindContext()
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
	ctx := newMockBindContext()

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
