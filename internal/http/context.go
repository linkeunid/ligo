package http

import "net/http"

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
}
