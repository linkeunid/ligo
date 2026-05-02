package ligo

// Package ligo provides HTTP routing abstractions and built-in utilities
// for request processing including Guards, Pipes, Interceptors, and Exception Filters.

import (
	"time"

	"github.com/linkeunid/ligo/internal/http"
)

// Router is the HTTP router interface that all router adapters must implement.
// It provides methods for registering routes, middleware, and managing the HTTP server.
type Router = http.Router

// HandlerFunc is the standard handler signature for route handlers.
// It receives a Context and returns an error.
type HandlerFunc = http.HandlerFunc

// Middleware is a function that wraps a handler to add pre/post processing.
// Middleware can modify the request, response, or short-circuit the handler chain.
type Middleware = http.Middleware

// Context wraps HTTP request/response for handlers, providing methods for
// accessing request data, binding bodies, and sending responses.
type Context = http.Context

// Controller defines how HTTP routes are registered for a module.
// Controllers receive dependencies via DI and register routes using the Router.
type Controller = http.Controller

// Guard determines if a request should proceed (authorization).
// A Guard returns (true, nil) to allow the request, or (false, error) to deny it.
type Guard = http.Guard

// Pipe transforms input data before it reaches the handler.
// Pipes are used for validation, parsing, and data transformation.
type Pipe = http.Pipe

// ValidatedBodyKey is the context key where ValidationPipe stores the validated body.
// Prefer ValidatedBody[T] over accessing this key directly.
const ValidatedBodyKey = http.ValidatedBodyKey

// ValidatedBody retrieves the validated body stored by ValidationPipe[T].
// Panics with a clear message if ValidationPipe was not added to the route.
//
// Example:
//
//	func (c *UserController) Create(ctx ligo.Context) error {
//	    input := ligo.ValidatedBody[CreateUserInput](ctx)
//	    // input is *CreateUserInput, guaranteed non-nil
//	}
func ValidatedBody[T any](ctx Context) *T {
	return http.ValidatedBody[T](ctx)
}

// Interceptor wraps the entire request/response cycle.
// Interceptors can modify the request before processing and the response after.
type Interceptor = http.Interceptor

// ExceptionFilter handles errors and converts them to HTTP responses.
// ExceptionFilters are called when handlers or other components return errors.
type ExceptionFilter = http.ExceptionFilter

// RouteBuilder provides fluent API for composing routes with Guards, Pipes,
// Interceptors, and ExceptionFilters using the builder pattern.
type RouteBuilder = http.RouteBuilder

// ChainRouter provides fluent chain methods for building routes.
// It allows chaining method calls for configuring routes.
type ChainRouter = http.ChainRouter

// NewChainRouter wraps a Router with chain methods.
// Example:
//
//	cr := ligo.NewChainRouter(router)
//	cr.GET("/", handler).Guard(authGuard).Handle()
func NewChainRouter(r Router) ChainRouter {
	return http.NewChainRouter(r)
}

// Built-in Pipes
//
// Pipes transform and validate request data before it reaches handlers.

// ValidationPipe validates a struct using struct tags.
// It uses the "validate" tag and requires the go-playground/validator package.
//
// Example:
//
//	type CreateUserInput struct {
//	    Name  string `validate:"required,min=3"`
//	    Email string `validate:"required,email"`
//	}
//
//	cr.POST("", c.Create).Pipe(ligo.ValidationPipe(&CreateUserInput{}))
func ValidationPipe[T any](v *T) Pipe {
	return http.ValidationPipe(v)
}

// ParseIntPipe parses a string parameter to int.
// Returns an error if the parameter cannot be parsed as an integer.
//
// Example:
//
//	cr.GET("/:id", c.Get).Pipe(ligo.ParseIntPipe("id"))
func ParseIntPipe(param string) Pipe {
	return http.ParseIntPipe(param)
}

// ParseBoolPipe parses a string parameter to bool.
// Accepts: "true", "false", "1", "0" (case-insensitive).
//
// Example:
//
//	cr.GET("/:active", c.Get).Pipe(ligo.ParseBoolPipe("active"))
func ParseBoolPipe(param string) Pipe {
	return http.ParseBoolPipe(param)
}

// UUIDPipe validates that a string parameter is a valid UUID format.
// Returns an error if the parameter is not a valid UUID.
//
// Example:
//
//	cr.GET("/:uuid", c.Get).Pipe(ligo.UUIDPipe("uuid"))
func UUIDPipe(param string) Pipe {
	return http.UUIDPipe(param)
}

// TrimPipe removes leading and trailing whitespace from a string parameter.
//
// Example:
//
//	cr.POST("", c.Create).Pipe(ligo.TrimPipe("name"))
func TrimPipe(param string) Pipe {
	return http.TrimPipe(param)
}

// Built-in Guards

// HasRole is an interface that types can implement for role checking.
type HasRole = http.HasRole

// RolesGuard creates a guard that checks if the user has one of the required roles.
// Usage: cr.GET("", c.List).Guard(ligo.RolesGuard("user", "admin"))
func RolesGuard(contextKey string, requiredRoles ...string) Guard {
	return http.RolesGuard(contextKey, requiredRoles...)
}

// AdminGuard is a convenience guard that checks for admin role.
// Usage: cr.DELETE("/:id", c.Delete).Guard(ligo.AdminGuard("user"))
func AdminGuard(contextKey string) Guard {
	return http.AdminGuard(contextKey)
}

// ThrottleGuard creates a rate-limiting guard.
// Usage: cr.POST("", c.Create).Guard(ligo.ThrottleGuard("ip", 10, time.Minute))
func ThrottleGuard(identifierKey string, maxRequests int, window time.Duration) Guard {
	return http.ThrottleGuard(identifierKey, maxRequests, window)
}

// Built-in Interceptors

// TimeoutInterceptor creates an interceptor that enforces a timeout.
// Usage: cr.GET("", c.List).Intercept(ligo.TimeoutInterceptor(5 * time.Second))
func TimeoutInterceptor(timeout time.Duration) Interceptor {
	return http.TimeoutInterceptor(timeout)
}

// LoggingInterceptor creates an interceptor that logs request details.
// Usage: cr.GET("", c.List).Intercept(ligo.LoggingInterceptor(func(start time.Time, ctx Context, err error) { ... }))
func LoggingInterceptor(logFunc func(start time.Time, ctx Context, err error)) Interceptor {
	return http.LoggingInterceptor(logFunc)
}
