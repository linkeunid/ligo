package logger

import (
	"log/slog"
	"os"
	"reflect"
)

// Context represents the internal component that generated the log.
type Context string

const (
	ContextApp         Context = "app"
	ContextDIContainer Context = "di.container"
	ContextRoutes      Context = "routes"
	ContextLifecycle   Context = "lifecycle"
	ContextMiddleware  Context = "middleware"
)

// Field is a key-value pair for structured logging.
type Field struct {
	Key   string
	Value any
}

// Type represents the logger output format.
type Type int

const (
	TypeText Type = iota
	TypeJSON
)

// Logger is the interface for framework logging.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

	LogWithContext(ctx Context, msg string, fields ...Field)
	SetDebug(enabled bool)
}

// SlogLogger wraps log/slog for Ligo's Logger interface.
type SlogLogger struct {
	handler slog.Handler
	logger  *slog.Logger
}

// LoggerOption configures the logger.
type LoggerOption func(*SlogLogger)

// WithJSON enables JSON output format.
func WithJSON() LoggerOption {
	return func(l *SlogLogger) {
		l.handler = slog.NewJSONHandler(os.Stderr, nil)
		l.logger = slog.New(l.handler)
	}
}

// WithText enables text output format (default).
func WithText() LoggerOption {
	return func(l *SlogLogger) {
		l.handler = slog.NewTextHandler(os.Stderr, nil)
		l.logger = slog.New(l.handler)
	}
}

// WithDebug enables debug logging.
func WithDebug(enabled bool) LoggerOption {
	return func(l *SlogLogger) {
		l.handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		l.logger = slog.New(l.handler)
	}
}

// WithProduction enables JSON logging (production mode).
func WithProduction() LoggerOption {
	return WithJSON()
}

// New creates a new slog-based logger. Default is text mode for development.
func New(opts ...LoggerOption) Logger {
	l := &SlogLogger{
		handler: slog.NewTextHandler(os.Stderr, nil),
	}
	l.logger = slog.New(l.handler)

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Debug logs a debug message.
func (l *SlogLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fieldsToSlogArgs(fields)...)
}

// Info logs an info message.
func (l *SlogLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fieldsToSlogArgs(fields)...)
}

// Warn logs a warning.
func (l *SlogLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fieldsToSlogArgs(fields)...)
}

// Error logs an error.
func (l *SlogLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fieldsToSlogArgs(fields)...)
}

// LogWithContext logs a message with a context tag.
func (l *SlogLogger) LogWithContext(ctx Context, msg string, fields ...Field) {
	allFields := append([]Field{{Key: "context", Value: string(ctx)}}, fields...)
	l.logger.Info(msg, fieldsToSlogArgs(allFields)...)
}

// SetDebug enables or disables debug logging.
func (l *SlogLogger) SetDebug(enabled bool) {
	level := slog.LevelInfo
	if enabled {
		level = slog.LevelDebug
	}

	// Recreate handler preserving text/JSON type
	if _, ok := l.handler.(*slog.JSONHandler); ok {
		l.handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		l.handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}
	l.logger = slog.New(l.handler)
}

func fieldsToSlogArgs(fields []Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		args = append(args, f.Key, f.Value)
	}
	return args
}

// ExtractProviderName extracts a clean name from a provider type or factory function.
func ExtractProviderName(fn any) string {
	if fn == nil {
		return "unknown"
	}

	typ := reflect.TypeOf(fn)
	if typ.Kind() == reflect.Func {
		if typ.NumOut() > 0 {
			retType := typ.Out(0)
			if retType.Kind() == reflect.Ptr {
				retType = retType.Elem()
			}
			return retType.Name()
		}
		return typ.Name()
	}

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}
