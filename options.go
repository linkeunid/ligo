package ligo

import (
	"time"

	"github.com/linkeunid/ligo/internal/core/logger"
)

type options struct {
	router             Router
	addr               string
	middlewares        []Middleware
	debug              bool
	logger             Logger
	gracefulShutdown  bool
	gracefulTimeout    time.Duration
	onStart            []LifecycleHook
	onStop             []LifecycleHook
}

// Logger is the interface for framework logging.
type Logger = logger.Logger

// LoggerType represents the logger output format.
type LoggerType = logger.Type

const (
	LoggerText = logger.TypeText
	LoggerJSON = logger.TypeJSON
)

// LoggerField is a key-value pair for structured logging.
type LoggerField = logger.Field

// NewLogger creates a new logger. Default is text mode for development.
func NewLogger(opts ...LoggerOption) Logger {
	return logger.New(opts...)
}

// LoggerOption configures the logger.
type LoggerOption = logger.LoggerOption

// WithLoggerText sets text output format.
func WithLoggerText() LoggerOption {
	return logger.WithText()
}

// WithLoggerJSON sets JSON output format.
func WithLoggerJSON() LoggerOption {
	return logger.WithJSON()
}

// WithLoggerProduction enables JSON logging.
func WithLoggerProduction() LoggerOption {
	return logger.WithProduction()
}

// WithLoggerDebug enables debug logging.
func WithLoggerDebug() LoggerOption {
	return logger.WithDebug(true)
}

// LifecycleHook is a function called during app lifecycle events.
type LifecycleHook func(ctx any) error

// Option configures the App.
type Option func(*options)

func defaultOptions() options {
	return options{
		addr: ":8080",
		logger: logger.New(),
		gracefulTimeout: 10 * time.Second,
	}
}

// WithRouter sets the HTTP router adapter.
func WithRouter(r Router) Option {
	return func(o *options) {
		o.router = r
	}
}

// WithAddr sets the server address.
func WithAddr(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

// WithDebug enables debug logging.
func WithDebug(debug bool) Option {
	return func(o *options) {
		o.debug = debug
		o.logger.SetDebug(debug)
	}
}

// WithJSON enables JSON logging mode (production).
func WithJSON() Option {
	return func(o *options) {
		o.logger = logger.New(logger.WithJSON())
	}
}

// WithMiddleware adds global middleware.
func WithMiddleware(mw ...Middleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, mw...)
	}
}

// WithLogger sets the logger.
func WithLogger(l Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}

// OnStart adds a hook to run on app startup.
func OnStart(hook LifecycleHook) Option {
	return func(o *options) {
		o.onStart = append(o.onStart, hook)
	}
}

// OnStop adds a hook to run on app shutdown.
func OnStop(hook LifecycleHook) Option {
	return func(o *options) {
		o.onStop = append(o.onStop, hook)
	}
}

// WithGracefulShutdown enables graceful shutdown on SIGINT/SIGTERM.
func WithGracefulShutdown(timeout time.Duration) Option {
	return func(o *options) {
		o.gracefulShutdown = true
		o.gracefulTimeout = timeout
	}
}