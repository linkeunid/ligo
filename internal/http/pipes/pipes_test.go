package pipes

import (
	"errors"
	"strings"
	"testing"
)

// mockCtx satisfies the local Context interface for pipe tests.
type mockCtx struct {
	bindBody   any
	bindErr    error
	values     map[string]any
	paramValue map[string]string
}

func newMockCtx() *mockCtx {
	return &mockCtx{
		values:     make(map[string]any),
		paramValue: make(map[string]string),
	}
}

func (m *mockCtx) Bind(v any) error {
	if m.bindErr != nil {
		return m.bindErr
	}
	// Caller passes &input; we want *input = bindBody if both are the same type.
	if m.bindBody != nil {
		// Copy the bindBody into *v using reflection-free indirection.
		switch target := v.(type) {
		case *bindStruct:
			if b, ok := m.bindBody.(bindStruct); ok {
				*target = b
			}
		}
	}
	return nil
}

func (m *mockCtx) Get(key string) any      { return m.values[key] }
func (m *mockCtx) Set(key string, val any) { m.values[key] = val }
func (m *mockCtx) Param(key string) string { return m.paramValue[key] }

type bindStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
}

func TestValidationPipe_StoresValidatedBody(t *testing.T) {
	ctx := newMockCtx()
	ctx.bindBody = bindStruct{Name: "alice", Email: "a@b.co"}

	pipe := ValidationPipe[bindStruct](nil)
	if err := pipe(ctx); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	got := ValidatedBody[bindStruct](ctx)
	if got.Name != "alice" || got.Email != "a@b.co" {
		t.Errorf("body not stored: %+v", got)
	}
}

func TestValidationPipe_BindFailureWrapsBadRequest(t *testing.T) {
	ctx := newMockCtx()
	ctx.bindErr = errors.New("malformed json")

	pipe := ValidationPipe[bindStruct](nil)
	err := pipe(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest in chain, got %v", err)
	}
	if !strings.Contains(err.Error(), "bind failed") {
		t.Errorf("expected 'bind failed' in message, got %q", err.Error())
	}
}

func TestValidationPipe_ValidationFailureWrapsBadRequest(t *testing.T) {
	ctx := newMockCtx()
	ctx.bindBody = bindStruct{Name: "", Email: "not-an-email"}

	pipe := ValidationPipe[bindStruct](nil)
	err := pipe(ctx)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest in chain, got %v", err)
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected 'validation failed' prefix, got %q", err.Error())
	}
}

func TestValidatedBody_PanicsWithClearMessage(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "ValidationPipe") {
			t.Errorf("panic message missing ValidationPipe hint: %q", msg)
		}
	}()

	ctx := newMockCtx()
	_ = ValidatedBody[bindStruct](ctx)
}

func TestParseIntPipe_Success(t *testing.T) {
	ctx := newMockCtx()
	ctx.paramValue["id"] = "42"

	pipe := ParseIntPipe("id")
	if err := pipe(ctx); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got, _ := ctx.values["id"].(int); got != 42 {
		t.Errorf("expected 42, got %v", ctx.values["id"])
	}
}

func TestParseIntPipe_InvalidWrapsBadRequest(t *testing.T) {
	ctx := newMockCtx()
	ctx.paramValue["id"] = "not-a-number"

	pipe := ParseIntPipe("id")
	err := pipe(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), `param "id"`) {
		t.Errorf("expected param name in message, got %q", err.Error())
	}
}

func TestParseBoolPipe_Success(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
		{"T", true},
		{"F", false},
	}
	for _, tc := range cases {
		ctx := newMockCtx()
		ctx.paramValue["flag"] = tc.in
		if err := ParseBoolPipe("flag")(ctx); err != nil {
			t.Errorf("%s: unexpected error %v", tc.in, err)
		}
		if got, _ := ctx.values["flag"].(bool); got != tc.want {
			t.Errorf("%s: expected %v, got %v", tc.in, tc.want, ctx.values["flag"])
		}
	}
}

func TestParseBoolPipe_InvalidWrapsBadRequest(t *testing.T) {
	ctx := newMockCtx()
	ctx.paramValue["flag"] = "maybe"

	err := ParseBoolPipe("flag")(ctx)
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestUUIDPipe_ValidUUID(t *testing.T) {
	ctx := newMockCtx()
	ctx.paramValue["id"] = "550e8400-e29b-41d4-a716-446655440000"

	if err := UUIDPipe("id")(ctx); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got, _ := ctx.values["id"].(string); got != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected stored UUID, got %v", ctx.values["id"])
	}
}

func TestUUIDPipe_InvalidWrapsBadRequest(t *testing.T) {
	ctx := newMockCtx()
	ctx.paramValue["id"] = "not-a-uuid"

	err := UUIDPipe("id")(ctx)
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), "valid UUID") {
		t.Errorf("expected 'valid UUID' hint, got %q", err.Error())
	}
}

func TestTrimPipe_StripsLeadingAndTrailingSpace(t *testing.T) {
	ctx := newMockCtx()
	ctx.paramValue["name"] = "   alice   "

	if err := TrimPipe("name")(ctx); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got, _ := ctx.values["name"].(string); got != "alice" {
		t.Errorf("expected 'alice', got %q", got)
	}
}

func TestTrimPipe_EmptyAfterTrim(t *testing.T) {
	ctx := newMockCtx()
	ctx.paramValue["name"] = "      "

	_ = TrimPipe("name")(ctx)
	if got, _ := ctx.values["name"].(string); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestTransformPipe_WrapsBadRequestOnError(t *testing.T) {
	ctx := newMockCtx()
	ctx.paramValue["x"] = "boom"

	custom := TransformPipe("x", func(string) (int, error) {
		return 0, errors.New("custom err")
	}, "custom")

	err := custom(ctx)
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), "custom pipe") {
		t.Errorf("expected pipe name in message, got %q", err.Error())
	}
}
