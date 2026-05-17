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

func TestMockContext_RequestAndResponseDefaults(t *testing.T) {
	m := NewMockContext()
	if m.Request() == nil {
		t.Error("default Request is nil")
	}
	if m.Response() == nil {
		t.Error("default Response is nil")
	}
	if m.RequestContext() == nil {
		t.Error("default RequestContext is nil")
	}
}

func TestMockContext_Param(t *testing.T) {
	m := NewMockContext()
	if got := m.Param("anything"); got != "" {
		t.Errorf("Param = %q, want \"\"", got)
	}
}

func TestMockContext_QueryHelpers(t *testing.T) {
	m := NewMockContext()
	// Default request is GET / — no query. Helpers must return defaults.
	if got := m.Query("k"); got != "" {
		t.Errorf("Query missing = %q", got)
	}
	if got := m.QueryDefault("k", "fallback"); got != "fallback" {
		t.Errorf("QueryDefault = %q", got)
	}
	if got := m.QueryInt("k", 7); got != 7 {
		t.Errorf("QueryInt = %d", got)
	}
}

func TestMockContext_SetGet(t *testing.T) {
	m := NewMockContext()
	m.Set("k", 42)
	if got := m.Get("k"); got != 42 {
		t.Errorf("Get = %v", got)
	}
	if got := m.Get("missing"); got != nil {
		t.Errorf("Get missing = %v", got)
	}
}

func TestMockContext_AllResponseHelpersReturnNil(t *testing.T) {
	m := NewMockContext()
	// Sweep every response helper to lock in the no-op contract.
	checks := []struct {
		name string
		fn   func() error
	}{
		{"OK", func() error { return m.OK(nil) }},
		{"Created", func() error { return m.Created(nil) }},
		{"Accepted", func() error { return m.Accepted(nil) }},
		{"NoContent", m.NoContent},
		{"List", func() error { return m.List(nil) }},
		{"Paginated", func() error { return m.Paginated(nil, 1, 10, 0) }},
		{"JSON", func() error { return m.JSON(200, nil) }},
		{"String", func() error { return m.String(200, "") }},
		{"BadRequest", func() error { return m.BadRequest() }},
		{"Unauthorized", func() error { return m.Unauthorized() }},
		{"Forbidden", func() error { return m.Forbidden() }},
		{"NotFound", func() error { return m.NotFound() }},
		{"MethodNotAllowed", func() error { return m.MethodNotAllowed() }},
		{"NotAcceptable", func() error { return m.NotAcceptable() }},
		{"RequestTimeout", func() error { return m.RequestTimeout() }},
		{"Conflict", func() error { return m.Conflict() }},
		{"Gone", func() error { return m.Gone() }},
		{"PreconditionFailed", func() error { return m.PreconditionFailed() }},
		{"PayloadTooLarge", func() error { return m.PayloadTooLarge() }},
		{"UnsupportedMediaType", func() error { return m.UnsupportedMediaType() }},
		{"UnprocessableEntity", func() error { return m.UnprocessableEntity() }},
		{"TooManyRequests", func() error { return m.TooManyRequests() }},
		{"InternalServerError", func() error { return m.InternalServerError() }},
		{"NotImplemented", func() error { return m.NotImplemented() }},
		{"BadGateway", func() error { return m.BadGateway() }},
		{"ServiceUnavailable", func() error { return m.ServiceUnavailable() }},
		{"GatewayTimeout", func() error { return m.GatewayTimeout() }},
		{"HTTPVersionNotSupported", func() error { return m.HTTPVersionNotSupported() }},
		{"Stream", func() error { return m.Stream(nil) }},
	}
	for _, c := range checks {
		if err := c.fn(); err != nil {
			t.Errorf("%s = %v, want nil", c.name, err)
		}
	}
}

func TestMockContext_Paginate(t *testing.T) {
	m := NewMockContext()
	q := m.Paginate(10, 100)
	// Defaults should be normalized: per_page absent → default 10.
	if q.PerPage != 10 {
		t.Errorf("PerPage = %d, want 10", q.PerPage)
	}
}

func TestMockLogger_AllLevels(t *testing.T) {
	l := NewMockLogger()
	l.Debug("d")
	l.Info("i")
	l.Warn("w")
	l.Error("e")
	l.LogWithContext("ctx", "msg")
	l.SetDebug(true) // no-op, just verify it does not panic

	logs := l.GetLogs()
	if len(logs) != 5 {
		t.Errorf("GetLogs count = %d, want 5", len(logs))
	}

	l.Clear()
	if len(l.GetLogs()) != 0 {
		t.Error("Clear did not empty logs")
	}
}
