package logger

import (
	"log/slog"
	"os"

	reflectutil "github.com/linkeunid/ligo/internal/reflect"
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
//
// handler and logger are written exactly once in New and treated as read-only
// afterward, so they need no synchronization. SetDebug mutates only the
// LevelVar, which is atomic by contract.
type SlogLogger struct {
	handler slog.Handler
	logger  *slog.Logger
	level   *slog.LevelVar
}

// loggerConfig accumulates option values before the handler is built.
// Building the handler once at the end of New avoids the prior trap where
// `WithJSON()` / `WithText()` / `WithDebug()` would overwrite each other's
// state depending on application order.
type loggerConfig struct {
	format Type
	level  *slog.LevelVar
}

// LoggerOption configures the logger.
type LoggerOption func(*loggerConfig)

// WithJSON enables JSON output format.
func WithJSON() LoggerOption {
	return func(c *loggerConfig) { c.format = TypeJSON }
}

// WithText enables text output format (default).
func WithText() LoggerOption {
	return func(c *loggerConfig) { c.format = TypeText }
}

// WithDebug enables debug logging.
func WithDebug(enabled bool) LoggerOption {
	return func(c *loggerConfig) {
		if enabled {
			c.level.Set(slog.LevelDebug)
		} else {
			c.level.Set(slog.LevelInfo)
		}
	}
}

// WithProduction enables JSON logging (production mode).
func WithProduction() LoggerOption {
	return WithJSON()
}

// New creates a new slog-based logger. Default is text mode for development.
func New(opts ...LoggerOption) Logger {
	cfg := &loggerConfig{
		format: TypeText,
		level:  new(slog.LevelVar), // defaults to slog.LevelInfo
	}
	for _, opt := range opts {
		opt(cfg)
	}

	handlerOpts := &slog.HandlerOptions{Level: cfg.level}
	var handler slog.Handler
	if cfg.format == TypeJSON {
		handler = slog.NewJSONHandler(os.Stderr, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, handlerOpts)
	}

	return &SlogLogger{
		handler: handler,
		logger:  slog.New(handler),
		level:   cfg.level,
	}
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

// SetDebug enables or disables debug logging. Safe for concurrent use:
// the underlying slog.LevelVar mutates atomically. No-op on loggers
// constructed without a LevelVar (e.g. manually composed test fixtures).
func (l *SlogLogger) SetDebug(enabled bool) {
	if l.level == nil {
		return
	}
	if enabled {
		l.level.Set(slog.LevelDebug)
	} else {
		l.level.Set(slog.LevelInfo)
	}
}

func fieldsToSlogArgs(fields []Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		args = append(args, f.Key, f.Value)
	}
	return args
}

// Noop returns a Logger that silently discards every call. Useful as a
// default for components that accept an optional logger.
func Noop() Logger { return noopLogger{} }

type noopLogger struct{}

func (noopLogger) Debug(string, ...Field)                   {}
func (noopLogger) Info(string, ...Field)                    {}
func (noopLogger) Warn(string, ...Field)                    {}
func (noopLogger) Error(string, ...Field)                   {}
func (noopLogger) LogWithContext(Context, string, ...Field) {}
func (noopLogger) SetDebug(bool)                            {}

// ExtractProviderName extracts a clean name from a provider type or factory function.
// Deprecated: Use reflectutil.ExtractTypeName instead for better consistency.
func ExtractProviderName(fn any) string {
	return reflectutil.ExtractTypeName(fn)
}
