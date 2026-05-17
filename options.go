package ligo

// Package ligo provides configuration options for the Ligo application,
// including router, middleware, logging, and lifecycle hooks.

import (
	"time"

	"github.com/linkeunid/ligo/internal/core/logger"
)

const (
	// DefaultPort is the default HTTP port for the server.
	DefaultPort = ":8080"
	// DefaultGracefulTimeout is the default timeout for graceful shutdown.
	DefaultGracefulTimeout = 10 * time.Second
)

type options struct {
	router           Router
	addr             string
	middlewares      []Middleware
	debug            bool
	logger           Logger
	gracefulShutdown bool
	gracefulTimeout  time.Duration
	onStart          []LifecycleHook
	onStop           []LifecycleHook
	autoPort         bool
	parallelHooks    bool
}

// Logger is the interface for framework logging.
type Logger = logger.Logger

// LoggerType represents the logger output format.
type LoggerType = logger.Type

const (
	// LoggerText enables human-readable text logging.
	LoggerText = logger.TypeText
	// LoggerJSON enables structured JSON logging.
	LoggerJSON = logger.TypeJSON
)

// LoggerField is a key-value pair for structured logging.
type LoggerField = logger.Field

// NewLogger creates a new logger. Default is text mode for development.
//
// Example:
//
//	logger := ligo.NewLogger(
//	    ligo.WithLoggerJSON(),
//	    ligo.WithLoggerDebug(),
//	)
func NewLogger(opts ...LoggerOption) Logger {
	return logger.New(opts...)
}

// LoggerOption configures the logger.
type LoggerOption = logger.LoggerOption

// WithLoggerText sets text output format (default).
func WithLoggerText() LoggerOption {
	return logger.WithText()
}

// WithLoggerJSON sets JSON output format for production.
func WithLoggerJSON() LoggerOption {
	return logger.WithJSON()
}

// WithLoggerProduction enables JSON logging (alias for WithLoggerJSON).
func WithLoggerProduction() LoggerOption {
	return logger.WithProduction()
}

// WithLoggerDebug enables debug logging.
func WithLoggerDebug() LoggerOption {
	return logger.WithDebug(true)
}

// LifecycleHook is a function called during app lifecycle events.
// The ctx parameter is context.Context for OnStart/OnStop hooks.
type LifecycleHook func(ctx any) error

// Option configures the App.
type Option func(*options)

func defaultOptions() options {
	return options{
		addr:            DefaultPort,
		logger:          logger.New(),
		gracefulTimeout: DefaultGracefulTimeout,
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

// NoopLogger returns a Logger that discards every call. Convenient for tests
// that want to bypass log spam without importing internal/core/logger.
func NoopLogger() Logger { return logger.Noop() }

// WithMiddleware adds global middleware.
func WithMiddleware(mw ...Middleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, mw...)
	}
}

// WithLogger overrides the framework logger. Useful in tests (pass
// ligo.NoopLogger() to silence startup/shutdown chatter) and for binaries
// that want to share a logger across the app and their own subsystems.
// nil is treated as "no override" — keeps the default constructed logger.
func WithLogger(l Logger) Option {
	return func(o *options) {
		if l != nil {
			o.logger = l
		}
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

// WithAutoPort enables automatic port increment if the default port is already in use.
func WithAutoPort() Option {
	return func(o *options) {
		o.autoPort = true
	}
}

// WithParallelHooks runs provider OnInit and OnBootstrap hooks concurrently.
// Default is sequential execution in registration order — opt in only when
// the application has many independent providers whose startup work is
// I/O-bound (DB pools, remote handshakes) and order does not matter.
//
// Parallel execution does not guarantee any ordering between hooks and
// surfaces a single aggregated error if any hook fails.
func WithParallelHooks() Option {
	return func(o *options) {
		o.parallelHooks = true
	}
}
