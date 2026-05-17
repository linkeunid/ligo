package http

import (
	"context"

	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/di"
)

// Router abstracts the HTTP router implementation.
type Router interface {
	Group(prefix string) Router
	Use(middleware ...Middleware)
	Handle(method, path string, handler HandlerFunc)
	Serve(addr string) error
}

// NullRouter is a no-op router used for non-HTTP applications.
// It allows controllers to be instantiated for lifecycle hooks without actual routing.
type NullRouter struct{}

// Group returns a new NullRouter (no-op for non-HTTP mode).
func (n *NullRouter) Group(prefix string) Router { return n }

// Use is a no-op for non-HTTP mode.
func (n *NullRouter) Use(middleware ...Middleware) {}

// Handle is a no-op for non-HTTP mode.
func (n *NullRouter) Handle(method, path string, handler HandlerFunc) {}

// Serve returns an error since NullRouter should not serve HTTP.
func (n *NullRouter) Serve(addr string) error {
	return context.Canceled
}

// RouteBuilder provides fluent API for composing routes with guards, pipes, interceptors, and filters.
type RouteBuilder interface {
	Guard(guards ...Guard) RouteBuilder
	Pipe(pipes ...Pipe) RouteBuilder
	Intercept(interceptors ...Interceptor) RouteBuilder
	Use(middleware ...Middleware) RouteBuilder
	Filter(filters ...ExceptionFilter) RouteBuilder
	Handle(handler ...HandlerFunc)
}

// Guard determines if a request should proceed (authorization).
type Guard func(ctx Context) (bool, error)

// Pipe transforms and validates request data before it reaches the handler.
// It receives the full request context so it can read path params, bind the
// request body, or perform any other context-aware transformation.
type Pipe func(ctx Context) error

// Interceptor wraps the entire request/response cycle (logging, caching, transformation).
type Interceptor func(ctx Context, next HandlerFunc) error

// ExceptionFilter handles errors and converts them to HTTP responses.
type ExceptionFilter func(error, Context) error

// SetLoggerRouter extends Router with optional logger support.
type SetLoggerRouter interface {
	Router
	SetLogger(logger.Logger)
}

// SetContainerRouter extends Router with DI container support for request-scoped DI.
type SetContainerRouter interface {
	Router
	SetContainer(*di.Container)
}

// GracefulServer extends Router with graceful shutdown capability.
type GracefulServer interface {
	Router
	Shutdown(ctx context.Context) error
}
