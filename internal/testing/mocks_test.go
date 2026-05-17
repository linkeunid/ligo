package testing

import (
	"errors"
	"strings"
	"testing"
)

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

func TestMockContext_Query(t *testing.T) {
	m := NewMockContext()
	// Default request is GET / — no query.
	if got := m.Query("k"); got != "" {
		t.Errorf("Query missing = %q", got)
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

func TestMockContext_RecordsJSON(t *testing.T) {
	m := NewMockContext()
	body := map[string]string{"hello": "world"}
	if err := m.JSON(201, body); err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	if m.LastJSONCode != 201 {
		t.Errorf("expected code 201, got %d", m.LastJSONCode)
	}
	got, ok := m.LastJSONBody.(map[string]string)
	if !ok || got["hello"] != "world" {
		t.Errorf("body not recorded: %#v", m.LastJSONBody)
	}
}

func TestMockContext_RecordsString(t *testing.T) {
	m := NewMockContext()
	if err := m.String(204, "no content"); err != nil {
		t.Fatalf("String returned error: %v", err)
	}
	if m.LastStringCode != 204 {
		t.Errorf("expected code 204, got %d", m.LastStringCode)
	}
	if m.LastStringBody != "no content" {
		t.Errorf("body not recorded: %q", m.LastStringBody)
	}
}

func TestMockContext_RecordsStream(t *testing.T) {
	m := NewMockContext()
	if err := m.Stream(strings.NewReader("payload")); err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if m.LastStreamBody != "payload" {
		t.Errorf("stream body not recorded: %q", m.LastStreamBody)
	}
}

func TestMockContext_JSONReturnsInjectedError(t *testing.T) {
	m := NewMockContext()
	m.JSONErr = errors.New("boom")
	if err := m.JSON(200, nil); err == nil || err.Error() != "boom" {
		t.Errorf("expected injected error, got %v", err)
	}
}
