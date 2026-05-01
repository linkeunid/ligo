package http

import (
	"context"

	"github.com/linkeunid/ligo/internal/core/logger"
)

// Router abstracts the HTTP router implementation.
type Router interface {
	Group(prefix string) Router
	Use(middleware ...Middleware)
	Handle(method, path string, handler HandlerFunc)
	Serve(addr string) error
}

// RouteBuilder provides fluent API for composing routes with guards, pipes, interceptors, and filters.
type RouteBuilder interface {
	Guard(guards ...Guard) RouteBuilder
	Pipe(pipes ...Pipe) RouteBuilder
	Intercept(interceptors ...Interceptor) RouteBuilder
	Use(middleware ...Middleware) RouteBuilder
	Filter(filters ...ExceptionFilter) RouteBuilder
	Handle(handler HandlerFunc)
}

// Guard determines if a request should proceed (authorization).
type Guard func(ctx Context) (bool, error)

// Pipe transforms input data before it reaches the handler (validation, parsing).
type Pipe func(any) (any, error)

// Interceptor wraps the entire request/response cycle (logging, caching, transformation).
type Interceptor func(ctx Context, next HandlerFunc) error

// ExceptionFilter handles errors and converts them to HTTP responses.
type ExceptionFilter func(error, Context) error

// SetLoggerRouter extends Router with optional logger support.
type SetLoggerRouter interface {
	Router
	SetLogger(logger.Logger)
}

// GracefulServer extends Router with graceful shutdown capability.
type GracefulServer interface {
	Router
	Shutdown(ctx context.Context) error
}
