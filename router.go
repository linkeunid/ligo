package ligo

import (
	"time"

	"github.com/linkeunid/ligo/internal/http"
)

// Router is the HTTP router interface.
type Router = http.Router

// HandlerFunc is the standard handler signature.
type HandlerFunc = http.HandlerFunc

// Middleware is a function that wraps a handler.
type Middleware = http.Middleware

// Context wraps HTTP request/response for handlers.
type Context = http.Context

// Controller defines how HTTP routes are registered for a module.
type Controller = http.Controller

// Guard determines if a request should proceed (authorization).
type Guard = http.Guard

// Pipe transforms input data before it reaches the handler.
type Pipe = http.Pipe

// Interceptor wraps the entire request/response cycle.
type Interceptor = http.Interceptor

// ExceptionFilter handles errors and converts them to HTTP responses.
type ExceptionFilter = http.ExceptionFilter

// RouteBuilder provides fluent API for composing routes.
type RouteBuilder = http.RouteBuilder

// ChainRouter provides fluent chain methods for building routes.
type ChainRouter = http.ChainRouter

// NewChainRouter wraps a Router with chain methods.
func NewChainRouter(r Router) ChainRouter {
	return http.NewChainRouter(r)
}

// Built-in Pipes

// ValidationPipe validates a struct using struct tags.
// Usage: cr.POST("", c.Create).Pipe(ligo.ValidationPipe(&CreateUserInput{}))
func ValidationPipe[T any](v *T) Pipe {
	return http.ValidationPipe(v)
}

// ParseIntPipe parses a string parameter to int.
// Usage: cr.GET("/:id", c.Get).Pipe(ligo.ParseIntPipe("id"))
func ParseIntPipe(param string) Pipe {
	return http.ParseIntPipe(param)
}

// ParseBoolPipe parses a string parameter to bool.
// Usage: cr.GET("/:active", c.Get).Pipe(ligo.ParseBoolPipe("active"))
func ParseBoolPipe(param string) Pipe {
	return http.ParseBoolPipe(param)
}

// UUIDPipe validates that a string is a valid UUID format.
// Usage: cr.GET("/:uuid", c.Get).Pipe(ligo.UUIDPipe("uuid"))
func UUIDPipe(param string) Pipe {
	return http.UUIDPipe(param)
}

// TrimPipe removes leading and trailing whitespace from a string.
// Usage: cr.POST("", c.Create).Pipe(ligo.TrimPipe("name"))
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
