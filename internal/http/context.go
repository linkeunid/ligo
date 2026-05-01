package http

import (
	"net/http"

	"github.com/linkeunid/ligo/internal/core/container"
)

// Controller defines how HTTP routes are registered for a module.
type Controller interface {
	Routes(r Router)
}

// HandlerFunc is the standard handler signature.
type HandlerFunc func(ctx Context) error

// Middleware is a function that wraps a handler.
type Middleware func(HandlerFunc) HandlerFunc

// Context wraps HTTP request/response for handlers.
type Context interface {
	Request() *http.Request
	Response() http.ResponseWriter
	Param(key string) string
	Bind(v any) error
	JSON(code int, v any) error
	String(code int, s string) error

	// Request-scoped data storage
	Set(key string, val any)
	Get(key string) any

	// Request-scoped DI container
	SetRequestContainer(c *container.Container)
	GetRequestContainer() *container.Container

	// HTTP response helpers
	OK(v any) error
	Created(v any) error
	NoContent() error
	BadRequest(msg string) error
	Unauthorized(msg string) error
	Forbidden(msg string) error
	NotFound(msg string) error
	Conflict(msg string) error
	InternalServerError(msg string) error
}
