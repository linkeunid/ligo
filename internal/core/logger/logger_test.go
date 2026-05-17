package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("New() returned nil")
	}

	_, ok := l.(*SlogLogger)
	if !ok {
		t.Error("New() did not return *SlogLogger")
	}
}

func TestNewWithText(t *testing.T) {
	l := New(WithText())
	if l == nil {
		t.Fatal("New(WithText()) returned nil")
	}
	sl, ok := l.(*SlogLogger)
	if !ok {
		t.Fatal("Not a SlogLogger")
	}
	if _, ok := sl.handler.(*slog.TextHandler); !ok {
		t.Error("WithText() did not set TextHandler")
	}
}

func TestNewWithJSON(t *testing.T) {
	l := New(WithJSON())
	if l == nil {
		t.Fatal("New(WithJSON()) returned nil")
	}
	sl, ok := l.(*SlogLogger)
	if !ok {
		t.Fatal("Not a SlogLogger")
	}
	if _, ok := sl.handler.(*slog.JSONHandler); !ok {
		t.Error("WithJSON() did not set JSONHandler")
	}
}

func TestNewWithProduction(t *testing.T) {
	l := New(WithProduction())
	sl, ok := l.(*SlogLogger)
	if !ok {
		t.Fatal("Not a SlogLogger")
	}
	if _, ok := sl.handler.(*slog.JSONHandler); !ok {
		t.Error("WithProduction() did not set JSONHandler")
	}
}

func TestLoggerLevels(t *testing.T) {
	tests := []struct {
		name string
		fn   func(Logger)
	}{
		{"Debug", func(l Logger) { l.Debug("test debug") }},
		{"Info", func(l Logger) { l.Info("test info") }},
		{"Warn", func(l Logger) { l.Warn("test warn") }},
		{"Error", func(l Logger) { l.Error("test error") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()
			tt.fn(l)
		})
	}
}

func TestLoggerWithFields(t *testing.T) {
	l := New()
	l.Info("test", Field{Key: "key1", Value: "value1"}, Field{Key: "key2", Value: 123})
	l.Debug("debug test", Field{Key: "debug", Value: true})
}

func TestLogWithContext(t *testing.T) {
	tests := []struct {
		name    string
		context Context
	}{
		{"App", ContextApp},
		{"DIContainer", ContextDIContainer},
		{"Routes", ContextRoutes},
		{"Lifecycle", ContextLifecycle},
		{"Middleware", ContextMiddleware},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New()
			l.LogWithContext(tt.context, "test message")
		})
	}
}

func TestSetDebug(t *testing.T) {
	l := New()

	l.SetDebug(true)
	l.Debug("debug message when enabled")

	l.SetDebug(false)
	l.Debug("debug message when disabled")
}

func TestExtractProviderName(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil", nil, "unknown"},
		{"string", "test", "string"},
		{"struct", struct{ Name string }{Name: "test"}, ""},
		{"pointer to struct", &struct{ Name string }{Name: "test"}, ""},
		{"function returning struct", func() *struct{ Name string } { return nil }, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractProviderName(tt.input)
			if tt.expected != "" && result != tt.expected {
				t.Errorf("ExtractProviderName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	var buf bytes.Buffer
	l := &SlogLogger{
		handler: slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}),
		logger:  slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}

	l.Info("test message", Field{Key: "key", Value: "value"})
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Output does not contain message: %v", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("Output does not contain field key: %v", output)
	}
}

func TestNoop_DoesNotPanic(t *testing.T) {
	l := Noop()
	l.Debug("d", Field{Key: "k", Value: "v"})
	l.Info("i")
	l.Warn("w")
	l.Error("e")
	l.LogWithContext(ContextApp, "ctx")
	l.SetDebug(true)
	l.SetDebug(false)
}

// Regression: WithJSON + WithDebug used to clobber each other depending
// on application order. Both orderings must now yield a JSON handler at
// debug level.
func TestOptionsCompose_JSONAndDebug(t *testing.T) {
	cases := []struct {
		name string
		opts []LoggerOption
	}{
		{"json then debug", []LoggerOption{WithJSON(), WithDebug(true)}},
		{"debug then json", []LoggerOption{WithDebug(true), WithJSON()}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := New(tc.opts...).(*SlogLogger)
			if _, ok := l.handler.(*slog.JSONHandler); !ok {
				t.Errorf("expected JSON handler, got %T", l.handler)
			}
			if l.level == nil {
				t.Fatal("expected non-nil LevelVar")
			}
			if got := l.level.Level(); got != slog.LevelDebug {
				t.Errorf("expected debug level, got %v", got)
			}
		})
	}
}

func TestOptionsCompose_TextDefaultAtInfo(t *testing.T) {
	l := New().(*SlogLogger)
	if _, ok := l.handler.(*slog.TextHandler); !ok {
		t.Errorf("expected default TextHandler, got %T", l.handler)
	}
	if got := l.level.Level(); got != slog.LevelInfo {
		t.Errorf("expected default info level, got %v", got)
	}
}

func TestSetDebug_NoHandlerRecreation(t *testing.T) {
	l := New().(*SlogLogger)
	before := l.handler
	l.SetDebug(true)
	if l.handler != before {
		t.Error("SetDebug should not recreate the handler")
	}
	if got := l.level.Level(); got != slog.LevelDebug {
		t.Errorf("expected debug level after SetDebug(true), got %v", got)
	}
	l.SetDebug(false)
	if got := l.level.Level(); got != slog.LevelInfo {
		t.Errorf("expected info level after SetDebug(false), got %v", got)
	}
}

func TestSetDebug_ConcurrentNoRace(t *testing.T) {
	l := New()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			l.SetDebug(i%2 == 0)
			l.Debug("msg")
			l.Info("msg")
		})
	}
	wg.Wait()
}

func TestSetDebug_NilLevelIsNoOp(t *testing.T) {
	// Manually-constructed fixture with no LevelVar should not panic.
	var buf bytes.Buffer
	l := &SlogLogger{
		handler: slog.NewTextHandler(&buf, nil),
		logger:  slog.New(slog.NewTextHandler(&buf, nil)),
	}
	l.SetDebug(true)
	l.SetDebug(false)
}

func TestFieldsToSlogArgs(t *testing.T) {
	fields := []Field{
		{Key: "string", Value: "value"},
		{Key: "int", Value: 123},
		{Key: "bool", Value: true},
	}

	args := fieldsToSlogArgs(fields)
	if len(args) != len(fields)*2 {
		t.Errorf("fieldsToSlogArgs() returned %d args, want %d", len(args), len(fields)*2)
	}
}
